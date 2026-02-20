package env

import (
	"path/filepath"
	"time"
)

// Options is e2e-only configuration.
// Keep this independent from pkg/slo (v1 legacy).
// Options는 e2e 전용 설정입니다.
// pkg/slo와 독립적으로 유지하세요 (v1 레거시).
type Options struct {
	Enabled      bool
	ArtifactsDir string
	RunID        string

	SkipCleanup            bool
	SkipCertManagerInstall bool

	TokenRequestTimeout time.Duration
}

// Validate checks configuration and applies defaults.
// Validate는 설정을 확인하고 기본값을 적용합니다.
func (o Options) Validate() Options {
	out := o
	if out.ArtifactsDir == "" {
		out.ArtifactsDir = "/tmp"
	}
	if out.TokenRequestTimeout == 0 {
		out.TokenRequestTimeout = 2 * time.Minute
	}
	return out
}

// SummaryPath returns the path for the summary file.
// SummaryPath는 요약 파일의 경로를 반환합니다.
func (o Options) SummaryPath(filename string) string {
	v := o.Validate()
	if filename == "" {
		filename = "sli-summary.json"
	}
	return filepath.Join(v.ArtifactsDir, filename)
}
