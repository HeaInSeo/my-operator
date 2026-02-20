package spec

import (
	"fmt"
	"strings"

	"github.com/yeongki/my-operator/pkg/slo/common/promkey"
)

// MetricRef identifies a metric input to an SLI.
// v3: simplest form uses a canonical Prometheus "text key" string.
// Example: controller_runtime_reconcile_total{result="success"}
// MetricRef는 SLI에 대한 메트릭 입력을 식별합니다.
// v3: 가장 단순한 형태는 정규 프로메테우스 "텍스트 키" 문자열을 사용합니다.
// 예: controller_runtime_reconcile_total{result="success"}
type MetricRef struct {
	Key   string
	Alias string // optional / 선택 사항
}

// UnsafePromKey creates a MetricRef from a raw string key.
// UnsafePromKey는 원시 문자열 키로부터 MetricRef를 생성합니다.
func UnsafePromKey(key string) MetricRef { return MetricRef{Key: key} }

// Labels represents a set of Prometheus labels.
// Labels는 프로메테우스 레이블 집합을 나타냅니다.
type Labels map[string]string

// PromMetric creates a MetricRef from a name and labels.
// PromMetric은 이름과 레이블로부터 MetricRef를 생성합니다.
func PromMetric(name string, labels Labels) MetricRef {
	return MetricRef{Key: promkey.Format(name, map[string]string(labels))}
}

// ComputeMode defines how to compute the SLI value.
// ComputeMode는 SLI 값을 계산하는 방법을 정의합니다.
type ComputeMode string

const (
	// ComputeSingle uses the start snapshot only.
	// ComputeSingle은 시작 스냅샷만 사용합니다.
	ComputeSingle ComputeMode = "single" // use start snapshot only (legacy v3) / 시작 스냅샷만 사용 (레거시 v3)
	// ComputeDelta uses the difference between end and start values.
	// ComputeDelta는 종료 값과 시작 값의 차이를 사용합니다.
	ComputeDelta ComputeMode = "delta" // end - start / 종료 - 시작

	// ComputeStart uses the start value.
	// ComputeStart는 시작 값을 사용합니다.
	ComputeStart ComputeMode = "start"
	// ComputeEnd uses the end value.
	// ComputeEnd는 종료 값을 사용합니다.
	ComputeEnd ComputeMode = "end"
)

// ComputeSpec describes how to compute the SLI.
// ComputeSpec은 SLI 계산 방법을 설명합니다.
type ComputeSpec struct {
	Mode ComputeMode
}

// Level defines the severity of a rule violation.
// Level은 규칙 위반의 심각도를 정의합니다.
type Level string

const (
	// LevelWarn indicates a warning.
	// LevelWarn은 경고를 나타냅니다.
	LevelWarn Level = "warn"
	// LevelFail indicates a failure.
	// LevelFail은 실패를 나타냅니다.
	LevelFail Level = "fail"
)

// Op defines the comparison operator.
// Op는 비교 연산자를 정의합니다.
type Op string

const (
	// OpLE is less than or equal.
	// OpLE는 작거나 같음입니다 (<=).
	OpLE Op = "<="
	// OpGE is greater than or equal.
	// OpGE는 크거나 같음입니다 (>=).
	OpGE Op = ">="
	// OpLT is less than.
	// OpLT는 작음입니다 (<).
	OpLT Op = "<"
	// OpGT is greater than.
	// OpGT는 큼입니다 (>).
	OpGT Op = ">"
	// OpEQ is equal.
	// OpEQ는 같음입니다 (==).
	OpEQ Op = "=="
)

// UnmarshalText decodes the operator from text.
// UnmarshalText는 텍스트에서 연산자를 디코딩합니다.
func (o *Op) UnmarshalText(text []byte) error {
	op, ok := NormalizeOp(string(text))
	if !ok {
		return fmt.Errorf("invalid op: %q", string(text))
	}
	*o = op
	return nil
}

// Rule is a tiny evaluation rule for v3.
// Rule은 v3를 위한 작은 평가 규칙입니다.
type Rule struct {
	Metric string  // usually "value" for v3
	Op     Op      // OpLE/OpGE/...
	Target float64 // threshold
	Level  Level   // LevelWarn | LevelFail
}

// JudgeSpec defines the rules for judging the SLI value.
// JudgeSpec은 SLI 값을 판단하기 위한 규칙을 정의합니다.
type JudgeSpec struct {
	Rules []Rule
}

// SLISpec is a declarative SLI definition.
// It is intentionally small in v3.
// SLISpec은 선언적 SLI 정의입니다.
// v3에서는 의도적으로 작게 유지됩니다.
type SLISpec struct {
	ID          string
	Title       string
	Unit        string
	Kind        string // "delta_counter" | "gauge" | "derived" (v3 minimal)
	Description string

	Inputs  []MetricRef
	Compute ComputeSpec
	Judge   *JudgeSpec
}

// NormalizeOp parses a string into an Op.
// NormalizeOp는 문자열을 Op로 파싱합니다.
func NormalizeOp(s string) (Op, bool) {
	t := strings.TrimSpace(strings.ToLower(s))

	switch t {
	case "<=", "=<":
		return OpLE, true
	case ">=", "=>":
		return OpGE, true
	case "<":
		return OpLT, true
	case ">":
		return OpGT, true
	case "==", "=":
		return OpEQ, true

	// 선택: 사람 친화 별칭
	case "le", "lte":
		return OpLE, true
	case "ge", "gte":
		return OpGE, true
	case "lt":
		return OpLT, true
	case "gt":
		return OpGT, true
	case "eq":
		return OpEQ, true
	case "\u2264": // ≤
		return OpLE, true
	case "\u2265": // ≥
		return OpGE, true
	default:
		return "", false
	}
}
