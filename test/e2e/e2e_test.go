package e2e

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/yeongki/my-operator/pkg/devutil"
	"github.com/yeongki/my-operator/pkg/kubeutil"
	"github.com/yeongki/my-operator/test/e2e/harness"
	e2eenv "github.com/yeongki/my-operator/test/e2e/internal/env"
)

const namespace = "my-operator-system"
const serviceAccountName = "my-operator-controller-manager"
const metricsServiceName = "my-operator-controller-manager-metrics-service"

var _ = Describe("Manager", Ordered, func() {
	var (
		cfg     e2eenv.Options
		token   string
		rootDir string
	)

	BeforeAll(func() {
		cfg = e2eenv.LoadOptions()
		By(fmt.Sprintf("ArtifactsDir=%q RunID=%q Enabled=%v", cfg.ArtifactsDir, cfg.RunID, cfg.Enabled))

		var err error
		rootDir, err = devutil.GetProjectDir()
		Expect(err).NotTo(HaveOccurred())

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		run := func(cmd *exec.Cmd, msg string) string {
			cmd.Dir = rootDir
			out, err := runner.Run(ctx, logger, cmd)
			Expect(err).NotTo(HaveOccurred(), msg)
			return out
		}

		By("creating manager namespace (idempotent)")
		// kubectl create ns can fail if already exists; we prefer apply-ish semantics.
		// Use: kubectl get ns || kubectl create ns
		cmd := exec.Command("bash", "-lc", fmt.Sprintf(`kubectl get ns %s >/dev/null 2>&1 || kubectl create ns %s`, namespace, namespace))
		run(cmd, "Failed to create namespace")

		By("labeling the namespace to enforce the security policy")
		cmd = exec.Command("kubectl", "label", "--overwrite", "ns", namespace,
			"pod-security.kubernetes.io/enforce=baseline")
		_, err = runner.Run(ctx, logger, cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to label namespace with security policy")

		By("installing CRDs")
		cmd = exec.Command("make", "install")
		run(cmd, "Failed to install CRDs")

		By("deploying the controller-manager")
		cmd = exec.Command("make", "deploy", fmt.Sprintf("IMG=%s", projectImage))
		run(cmd, "Failed to deploy the controller-manager")

		By("ensuring metrics reader RBAC for controller-manager SA (idempotent)")
		Expect(kubeutil.ApplyClusterRoleBinding(
			ctx, logger, runner,
			"my-operator-e2e-metrics-reader",
			"my-operator-metrics-reader",
			namespace,
			serviceAccountName,
		)).To(Succeed())
	})

	AfterAll(func() {
		if cfg.SkipCleanup {
			By("E2E_SKIP_CLEANUP enabled: skipping cleanup")
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		By("best-effort: cleaning up curl-metrics pods for metrics")
		cleanupCurlMetricsPods(namespace)

		By("undeploying the controller-manager (best-effort)")
		cmd := exec.Command("make", "undeploy")
		cmd.Dir = rootDir
		_, _ = runner.Run(ctx, logger, cmd)

		By("uninstalling CRDs (best-effort)")
		cmd = exec.Command("make", "uninstall")
		cmd.Dir = rootDir
		_, _ = runner.Run(ctx, logger, cmd)

		By("removing manager namespace (best-effort)")
		cmd = exec.Command("kubectl", "delete", "ns", namespace)
		_, _ = runner.Run(ctx, logger, cmd)
	})

	BeforeEach(func() {
		ctx, cancel := context.WithTimeout(context.Background(), cfg.TokenRequestTimeout)
		defer cancel()

		By("requesting service account token")
		t, err := kubeutil.ServiceAccountToken(ctx, logger, runner, namespace, serviceAccountName)
		Expect(err).NotTo(HaveOccurred())
		Expect(t).NotTo(BeEmpty())
		token = t

		By("waiting controller-manager ready")
		waitControllerManagerReady(namespace)

		By("waiting metrics service endpoints ready")
		waitServiceHasEndpoints(namespace, metricsServiceName)
	})

	harness.Attach(
		func() harness.HarnessDeps {
			return harness.HarnessDeps{
				ArtifactsDir: cfg.ArtifactsDir,
				Suite:        "e2e",
				TestCase:     "",
				RunID:        cfg.RunID,
				Enabled:      cfg.Enabled,
			}
		},
		func() harness.FetchDeps {
			return harness.FetchDeps{
				Namespace:          namespace,
				Token:              token,
				MetricsServiceName: metricsServiceName,
				ServiceAccountName: serviceAccountName,
			}
		},
		harness.DefaultV3Specs,
		harness.CurlPodFns{
			RunCurlMetricsOnce:  runCurlMetricsOnce,
			WaitCurlMetricsDone: waitCurlMetricsDone,
			CurlMetricsLogs:     curlMetricsLogs,
			DeletePodNoWait:     deletePodNoWait,
		},
	)

	It("should ensure the metrics endpoint is serving metrics", func() {
		By("scraping /metrics via curl pod")

		podName, err := runCurlMetricsOnce(namespace, token, metricsServiceName, serviceAccountName)
		Expect(err).NotTo(HaveOccurred())

		waitCurlMetricsDone(namespace, podName)

		text, err := curlMetricsLogs(namespace, podName)
		_ = deletePodNoWait(namespace, podName)
		Expect(err).NotTo(HaveOccurred())

		if !strings.Contains(text, "controller_runtime_reconcile_total") {
			head := text
			if len(head) > 800 {
				head = head[:800]
			}
			logger.Logf("metrics text head:\n%s", head)
		}

		Expect(text).To(ContainSubstring("controller_runtime_reconcile_total"))
		By(fmt.Sprintf("done (timeout=%s)", 2*time.Minute))
	})
})
