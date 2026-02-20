package e2e

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/yeongki/my-operator/pkg/devutil"
	"github.com/yeongki/my-operator/pkg/kubeutil"
	"github.com/yeongki/my-operator/pkg/slo"
	"github.com/yeongki/my-operator/test/e2e/e2eutil"
)

var (
	// Optional Environment Variables:
	// - CERT_MANAGER_INSTALL_SKIP=true: Skips CertManager installation during test setup.
	// 선택적 환경 변수:
	// - CERT_MANAGER_INSTALL_SKIP=true: 테스트 설정 중 CertManager 설치를 건너뜁니다.
	skipCertManagerInstall = os.Getenv("CERT_MANAGER_INSTALL_SKIP") == "true"

	// isCertManagerAlreadyInstalled will be set true when CertManager CRDs are found on the cluster.
	// isCertManagerAlreadyInstalled는 CertManager CRD가 클러스터에서 발견되면 true로 설정됩니다.
	isCertManagerAlreadyInstalled = false

	// projectImage is the name of the image which will be built and loaded with the code source changes to be tested.
	// projectImage는 테스트할 코드 소스 변경 사항으로 빌드되고 로드될 이미지의 이름입니다.
	projectImage = "example.com/my-operator:v0.0.1"

	// logger is the suite logger. It is always safe (nil -> no-op).
	// logger는 스위트 로거입니다. 항상 안전합니다 (nil -> no-op).
	logger = slo.NewLogger(e2eutil.GinkgoLog)

	// runner is used by kubeutil/devutil helpers (context-aware).
	// runner는 kubeutil/devutil 헬퍼(컨텍스트 인식)에서 사용됩니다.
	runner kubeutil.CmdRunner = kubeutil.DefaultRunner{}
	// useExistingCluster determines whether to use an existing cluster or provision Kind.
	// useExistingCluster는 기존 클러스터를 사용할지 아니면 Kind를 프로비저닝할지 결정합니다.
	useExistingCluster = os.Getenv("USE_EXISTING_CLUSTER") == "true"
)

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	logger.Logf("Starting my-operator integration test suite")
	RunSpecs(t, "e2e suite")
}

var _ = BeforeSuite(func() {
	// A reasonable default guard for setup steps.
	// Individual kubectl commands also have their own timeouts (e.g. kubectl wait --timeout).
	// 설정 단계를 위한 합리적인 기본 가드입니다.
	// 개별 kubectl 명령어들도 자체적인 타임아웃을 가지고 있습니다 (예: kubectl wait --timeout).
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	if useExistingCluster {
		logger.Logf("USE_EXISTING_CLUSTER=true: skipping Kind cluster image build and load")
	} else {
		By("building the manager(Operator) image")
		root, err := devutil.GetProjectDir()
		Expect(err).NotTo(HaveOccurred())

		cmd := exec.Command("make", "docker-build", fmt.Sprintf("IMG=%s", projectImage))
		cmd.Dir = root

		_, err = runner.Run(ctx, logger, cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to build the manager(Operator) image")

		By("loading the manager(Operator) image on Kind")
		Expect(devutil.LoadImageToKindClusterWithName(ctx, logger, runner, projectImage)).
			To(Succeed(), "Failed to load the manager(Operator) image into Kind")
	}

	// Setup CertManager before the suite if not skipped and if not already installed.
	if skipCertManagerInstall {
		logger.Logf("CERT_MANAGER_INSTALL_SKIP=true: skipping cert-manager setup")
		return
	}

	By("checking if cert-manager is installed already")
	isCertManagerAlreadyInstalled = kubeutil.IsCertManagerCRDsInstalled(ctx, logger, runner)
	if isCertManagerAlreadyInstalled {
		logger.Logf("WARNING: cert-manager is already installed; skipping installation")
		return
	}

	By("installing cert-manager")
	Expect(kubeutil.InstallCertManager(ctx, logger, runner)).
		To(Succeed(), "Failed to install cert-manager")
})

var _ = AfterSuite(func() {
	if skipCertManagerInstall || isCertManagerAlreadyInstalled {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	By("uninstalling cert-manager (best-effort)")
	if err := kubeutil.UninstallCertManager(ctx, logger, runner); err != nil {
		warnf("failed to uninstall cert-manager: %v", err)
	}
})

func warnf(format string, args ...any) {
	logger.Logf("WARNING: "+format, args...)
}
