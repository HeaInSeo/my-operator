package e2eutil

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/yeongki/my-operator/pkg/devutil"
)

// TODO 일단 생각하기.

// Logger is the minimal contract needed by ApplyTemplate.
// Logger는 ApplyTemplate에 필요한 최소한의 계약입니다.
type Logger interface {
	Logf(format string, args ...any)
}

// Runner is the minimal contract needed by ApplyTemplate.
// It matches the runner you already use: runner.Run(ctx, logger, cmd).
// Runner는 ApplyTemplate에 필요한 최소한의 계약입니다.
// 이미 사용 중인 runner와 일치합니다: runner.Run(ctx, logger, cmd).
type Runner interface {
	Run(ctx context.Context, logger Logger, cmd *exec.Cmd) (string, error)
}

// ApplyTemplate renders a manifest template file and applies it via `kubectl apply -f -`.
// - rootDir: repo root (used for template read + cmd.Dir)
// - relPath: template path relative to rootDir (e.g., "test/e2e/manifests/namespace.tmpl.yaml.gotmpl")
// - data: template data (struct/map)
// Returns kubectl stdout for debugging.
// ApplyTemplate renders a manifest template file and applies it via `kubectl apply -f -`.
// - rootDir: repo root (used for template read + cmd.Dir)
// - relPath: template path relative to rootDir
// - data: template data (struct/map)
// Returns kubectl stdout for debugging.
// ApplyTemplate은 매니페스트 템플릿 파일을 렌더링하고 `kubectl apply -f -`를 통해 적용합니다.
// - rootDir: 리포지토리 루트 (템플릿 읽기 + cmd.Dir에 사용됨)
// - relPath: rootDir에 대한 템플릿 상대 경로
// - data: 템플릿 데이터 (구조체/맵)
// 디버깅을 위해 kubectl stdout을 반환합니다.
func ApplyTemplate(
	ctx context.Context,
	rootDir string,
	relPath string,
	data any,
	runner Runner,
	logger Logger,
) (string, error) {
	manifest, err := devutil.RenderTemplateFileString(rootDir, relPath, data)
	if err != nil {
		return "", fmt.Errorf("render template %q: %w", relPath, err)
	}

	cmd := exec.Command("kubectl", "apply", "-f", "-")
	cmd.Dir = rootDir
	cmd.Stdin = strings.NewReader(manifest)

	out, err := runner.Run(ctx, logger, cmd)
	if err != nil {
		return out, fmt.Errorf("kubectl apply %q: %w", relPath, err)
	}
	return out, nil
}
