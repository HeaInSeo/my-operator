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

	"github.com/yeongki/my-operator/pkg/slo/fetch/curlpod"

	"github.com/yeongki/my-operator/test/e2e/harness"
	e2eenv "github.com/yeongki/my-operator/test/e2e/internal/env"
	"github.com/yeongki/my-operator/test/e2e/manifests"
)

// TODO 이거 따로 빼야 함.
const namespace = "my-operator-system"
const serviceAccountName = "my-operator-controller-manager"
const metricsServiceName = "my-operator-controller-manager-metrics-service"

var _ = Describe("Manager", Ordered, func() {
	var (
		// TODO 추후 런타임에 쓰이는 정보들, 초기 설정에 관련된 정보들, 계측에필요한 설정등은 정리는 했지만 문서로 만들어 놓자.
		cfg     e2eenv.Options
		rootDir string

		cm *curlpod.Client

		// shared per test
		metricsToken string
		metricsPod   *curlpod.CurlPod
		//token   string
	)

	BeforeAll(func() {
		cfg = e2eenv.LoadOptions()
		By(fmt.Sprintf("ArtifactsDir=%q RunID=%q Enabled=%v", cfg.ArtifactsDir, cfg.RunID, cfg.Enabled))

		var err error
		rootDir, err = devutil.GetProjectDir()
		Expect(err).NotTo(HaveOccurred())

		cm = curlpod.New(logger, runner)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		// TODO e2eutil 로 빼자.
		run := func(cmd *exec.Cmd, msg string) string {
			cmd.Dir = rootDir
			out, err := runner.Run(ctx, logger, cmd)
			Expect(err).NotTo(HaveOccurred(), msg)
			return out
		}

		By("Creating manager namespace with baseline security enforcement")
		//		nsManifest := fmt.Sprintf(`apiVersion: v1
		// kind: Namespace
		// metadata:
		//   name: %s
		// `, namespace)
		// TODO apply.go 에서 ApplyTemplate 적용할 지 고민중
		nsManifest, err := devutil.RenderTemplateFileString(
			rootDir,
			"test/e2e/manifests/namespace.tmpl.yaml.gotmpl",
			manifests.NamespaceData{Namespace: namespace},
		)
		Expect(err).NotTo(HaveOccurred())

		cmd := exec.Command("kubectl", "apply", "-f", "-")
		cmd.Dir = rootDir
		cmd.Stdin = strings.NewReader(nsManifest)
		run(cmd, "Failed to apply namespace with security policy")

		//By("labeling the namespace to enforce the security policy")
		//cmd = exec.Command("kubectl", "label", "--overwrite", "ns", namespace, "pod-security.kubernetes.io/enforce=baseline")
		//cmd.Dir = rootDir
		//run(cmd, "Failed to label namespace with security policy")

		By("installing CRDs")
		run(exec.Command("make", "install"), "Failed to install CRDs")

		By("deploying the controller-manager")
		run(exec.Command("make", "deploy", fmt.Sprintf("IMG=%s", projectImage)), "Failed to deploy the controller-manager")

		// TODO 추후 ApplyClusterRoleBinding 이걸 감싸서 구현할 수도 있는데 고민 중.
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
		// TODO 10*time.Minute 따로 빼자.
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		By("best-effort: cleaning up curl-metrics pods")
		_ = cm.CleanupByLabel(ctx, namespace)
		// TODO 기본 Makefile 에 대한 의존성이 생기지만 무시해도 될듯 한데, ????
		By("un-deploying the controller-manager (best-effort)")
		cmd := exec.Command("make", "undeploy")
		cmd.Dir = rootDir
		_, _ = runner.Run(ctx, logger, cmd)
		// TODO 기본 Makefile 에 대한 의존성이 생기지만 무시해도 될듯 한데, ????
		By("uninstalling CRDs (best-effort)")
		cmd = exec.Command("make", "uninstall")
		cmd.Dir = rootDir
		_, _ = runner.Run(ctx, logger, cmd)
		// TODO curlmetrics.go 사용하자.
		By("removing manager namespace (best-effort)")
		cmd = exec.Command("kubectl", "delete", "ns", namespace, "--ignore-not-found=true")
		cmd.Dir = rootDir
		_, _ = runner.Run(ctx, logger, cmd)
	})

	// TODO opts *WaitOptions 로 할지 고민 중 TODO: 5*time.Minute 따로 빼자.
	BeforeEach(func() {
		waitCtx, waitCancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer waitCancel()

		opts := kubeutil.WaitOptions{}

		By("waiting controller-manager ready")
		Expect(kubeutil.WaitControllerManagerReady(waitCtx, logger, runner, namespace, opts)).To(Succeed())

		By("waiting metrics service endpoints ready")
		Expect(kubeutil.WaitServiceHasEndpoints(waitCtx, logger, runner, namespace, metricsServiceName, opts)).To(Succeed())

		// ---- shared token + curlpod (used by BOTH harness + It) ----
		tokCtx, cancel := context.WithTimeout(context.Background(), cfg.TokenRequestTimeout)
		defer cancel()

		By("requesting service account token (shared)")
		t, err := kubeutil.ServiceAccountToken(tokCtx, logger, runner, namespace, serviceAccountName)
		Expect(err).NotTo(HaveOccurred())
		Expect(t).NotTo(BeEmpty())
		metricsToken = t

		metricsPod = &curlpod.CurlPod{
			Client:             cm,
			Namespace:          namespace,
			MetricsServiceName: metricsServiceName,
			ServiceAccountName: serviceAccountName,
			Token:              metricsToken,
			// Image / ServiceURLFormat override 필요하면 여기서 지정
		}

		// NOTE:
		// - 아래 Fetcher 타입은 제안한 방식대로 pkg/slo/fetch/curlpod/fetcher.go로 빼는 게 정석.
		// - 아직 없으면, 일단 harness 내부 default fetcher를 유지하거나,
		//   test-side adapter로 fetcher를 임시 구현해도 됨.
		// metricsFetcher = &curlpod.Fetcher{
		// 	Pod:               metricsPod,
		// 	AggregateNameOnly: true,
		// 	// timeouts override 필요하면 여기서
		// }
		// 일단 이렇게
	})

	// Use V4 Harness (Standardized)
	// V4 하니스 사용 (표준화됨)
	_, err := harness.Attach(func() harness.SessionConfig {
		// NOTE: token 발급/스크랩 로직은 BeforeEach에서 공유됨.
		// tokCtx, cancel := context.WithTimeout(context.Background(), cfg.TokenRequestTimeout)
		// defer cancel()

		// By("requesting service account token (for harness)")
		// t, err := kubeutil.ServiceAccountToken(tokCtx, logger, runner, namespace, serviceAccountName)
		// Expect(err).NotTo(HaveOccurred())
		// Expect(t).NotTo(BeEmpty())

		return harness.SessionConfig{
			// TODO Enabled 지울지 고민하자. 일단 주석처리함.
			//Enabled: 			cfg.Enabled,
			Namespace:          namespace,
			MetricsServiceName: metricsServiceName,
			TestCase:           "", // Auto-filled by harness
			Suite:              "e2e",

			RunID: cfg.RunID,
			//ServiceAccountName: serviceAccountName,
			//Token:              t,
			ArtifactsDir: cfg.ArtifactsDir,
			// TODO 일단 이렇게 주석처리함. 잘 봐야 함.
			//Fetcher: metricsFetcher,

			// TODO(태그): 런 상관관계(correlation) 분석을 위해 실행 메타 태그를 추가한다.
			// 예: git commit SHA, kind cluster name, controller image tag, k8s version, CI run id 등
			// Tags: map[string]string{
			// 	"commit":  "",
			// 	"cluster": "",
			// 	"image":   "",
			// },
		}
	})
	Expect(err).NotTo(HaveOccurred())

	It("should ensure the metrics endpoint is serving metrics", func() {
		By("scraping /metrics via curl pod")

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		text, err := metricsPod.Run(ctx, 5*time.Minute, 2*time.Minute)
		Expect(err).NotTo(HaveOccurred())

		if !strings.Contains(text, "controller_runtime_reconcile_total") {
			head := text
			if len(head) > 800 {
				head = head[:800]
			}
			logger.Logf("metrics text head:\n%s", head)
		}

		Expect(text).To(ContainSubstring("controller_runtime_reconcile_total"))
		By("done")
	})
})
