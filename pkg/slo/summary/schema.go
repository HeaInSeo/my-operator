package summary

import "time"

// Status is the normalized evaluation status for an SLIResult.
// Status는 SLIResult의 정규화된 평가 상태입니다.
type Status string

const (
	// StatusPass indicates success.
	// StatusPass는 성공을 나타냅니다.
	StatusPass Status = "pass"
	// StatusWarn indicates a warning.
	// StatusWarn은 경고를 나타냅니다.
	StatusWarn Status = "warn"
	// StatusFail indicates a failure.
	// StatusFail은 실패를 나타냅니다.
	StatusFail Status = "fail"
	// StatusSkip indicates the check was skipped.
	// StatusSkip은 체크가 건너뛰어졌음을 나타냅니다.
	StatusSkip Status = "skip"
)

// Summary is the contract output. All measurement methods must converge to this schema.
// Summary는 계약 출력입니다. 모든 측정 방식은 이 스키마로 수렴해야 합니다.
type Summary struct {
	SchemaVersion string    `json:"schemaVersion"`
	GeneratedAt   time.Time `json:"generatedAt"`

	Config RunConfig `json:"config"`

	Results  []SLIResult `json:"results"`
	Warnings []string    `json:"warnings,omitempty"`
}

// RunConfig is embedded in the summary (so analysis tools can be method-agnostic).
// RunConfig는 Summary에 포함됩니다 (분석 도구가 측정 방식에 구애받지 않도록 함).
type RunConfig struct {
	RunID      string            `json:"runId,omitempty"`
	StartedAt  time.Time         `json:"startedAt"`
	FinishedAt time.Time         `json:"finishedAt"`
	Mode       RunMode           `json:"mode"`
	Tags       map[string]string `json:"tags,omitempty"`
	Format     string            `json:"format,omitempty"`

	// EvidencePaths points to raw artifacts (optional).
	// EvidencePaths는 원시 아티팩트를 가리킵니다 (선택 사항).
	EvidencePaths map[string]string `json:"evidencePaths,omitempty"`
}

// RunMode describes how the run was executed.
// RunMode는 실행이 어떻게 수행되었는지를 설명합니다.
type RunMode struct {
	Location string `json:"location"` // "inside" | "outside"
	Trigger  string `json:"trigger"`  // "none" | "annotation"
}

// SLIResult captures the evaluation result of a single SLI.
// SLIResult는 단일 SLI의 평가 결과를 캡처합니다.
type SLIResult struct {
	ID          string `json:"id"`
	Title       string `json:"title,omitempty"`
	Unit        string `json:"unit,omitempty"`
	Kind        string `json:"kind,omitempty"`
	Description string `json:"description,omitempty"`

	// v3: a single numeric result. Future: Fields for p50/p99 etc.
	// v3: 단일 수치 결과. 향후: p50/p99 등의 필드 추가 예정.
	Value  *float64           `json:"value,omitempty"`
	Fields map[string]float64 `json:"fields,omitempty"`

	Status Status `json:"status"` // "pass" | "warn" | "fail" | "skip"

	Reason string `json:"reason,omitempty"`

	InputsUsed    []string `json:"inputsUsed,omitempty"`
	InputsMissing []string `json:"inputsMissing,omitempty"`
}

// EnsureFormat sets the format hint (defaulting to v4) while preserving schemaVersion.
// EnsureFormat은 schemaVersion을 보존하면서 포맷 힌트(기본값 v4)를 설정합니다.
func EnsureFormat(config map[string]any) map[string]any {
	if config == nil {
		config = map[string]any{}
	}
	if _, ok := config["format"]; !ok {
		config["format"] = "v4"
	}
	return config
}
