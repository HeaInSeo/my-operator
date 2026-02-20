package kubeutil

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/yeongki/my-operator/pkg/slo"
)

// ApplyClusterRoleBinding applies a ClusterRoleBinding in an idempotent way (kubectl apply).
// - logger may be nil (no-op).
// - r may be nil (uses DefaultRunner).
//
// TODO(security): Reduce YAML-injection risk by building a typed struct and marshaling
// (e.g. struct -> YAML/JSON), instead of fmt.Sprintf string templating.
// Even if we keep `kubectl apply`, struct->marshal makes input handling safer.
// ApplyClusterRoleBinding은 ClusterRoleBinding을 멱등원(idempotent) 방식(kubectl apply)으로 적용합니다.
// - logger는 nil일 수 있습니다 (no-op).
// - r은 nil일 수 있습니다 (DefaultRunner 사용).
//
// TODO(security): fmt.Sprintf 문자열 템플릿 대신, 타입이 지정된 구조체를 만들고
// 마샬링(예: struct -> YAML/JSON)하여 YAML 주입 위험을 줄이세요.
// `kubectl apply`를 계속 사용하더라도, struct->marshal 방식이 입력 처리에 더 안전합니다.
// Original:
// func ApplyClusterRoleBinding(ctx context.Context, logger slo.Logger, r CmdRunner,
//
//	name string, clusterRole string, ns string, sa string) error {
func ApplyClusterRoleBinding(
	ctx context.Context,
	logger slo.Logger,
	r CmdRunner,
	name string,
	clusterRole string,
	ns string,
	sa string,
) error {
	logger = slo.NewLogger(logger)
	if r == nil {
		r = DefaultRunner{}
	}

	logger.Logf(
		"apply ClusterRoleBinding name=%q role=%q sa=%s/%s",
		name,
		clusterRole,
		ns,
		sa,
	)

	manifest := fmt.Sprintf(`apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: %s
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: %s
subjects:
- kind: ServiceAccount
  name: %s
  namespace: %s
`, name, clusterRole, sa, ns)

	cmd := exec.Command("kubectl", "apply", "-f", "-")
	cmd.Stdin = strings.NewReader(manifest)

	stdout, err := r.Run(ctx, logger, cmd)

	if s := strings.TrimSpace(stdout); s != "" {
		logger.Logf("%s", strings.TrimRight(s, "\n"))
	}
	if err != nil {
		return fmt.Errorf("kubectl apply clusterrolebinding failed: %w", err)
	}
	return nil
}
