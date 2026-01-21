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
func ApplyClusterRoleBinding(ctx context.Context, logger slo.Logger, r CmdRunner, name, clusterRole, ns, sa string) error {
	logger = slo.NewLogger(logger)
	if r == nil {
		r = DefaultRunner{}
	}

	logger.Logf("apply ClusterRoleBinding name=%q role=%q sa=%s/%s", name, clusterRole, ns, sa)

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
