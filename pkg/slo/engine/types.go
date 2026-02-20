package engine

import (
	"time"

	"github.com/yeongki/my-operator/pkg/slo/spec"
)

// RunMode는 측정의 실행 컨텍스트를 정의함.
type RunMode struct {
	Location string // "inside" | "outside"
	Trigger  string // "none" | "annotation"
}

// RunConfig는 단일 실행별 설정을 담는다.
type RunConfig struct {
	RunID      string
	StartedAt  time.Time
	FinishedAt time.Time
	Mode       RunMode

	Tags          map[string]string
	Format        string
	EvidencePaths map[string]string
}

// ExecuteRequest는 SLO 체크 실행에 필요한 모든 데이터를 포함한다.
type ExecuteRequest struct {
	Config  RunConfig
	Specs   []spec.SLISpec // core input: 직접 주입
	OutPath string
	// 호환성/편의용: 레지스트리를 쓰는 호출자를 위해 남길 수 있음, 일단 주석처리함.
	// OutputPath is where the summary is written.
}

// MeasurementMethod는 측정 방식을 정의합니다 (표준화됨).
type MeasurementMethod string

const (
	// InsideSnapshot은 클러스터 내부에서 메트릭 스냅샷을 사용하여 SLI를 측정한다.
	InsideSnapshot MeasurementMethod = "InsideSnapshot"

	// InsideAnnotation은 이후 단계를 위해 예약되어 있다.
	InsideAnnotation MeasurementMethod = "InsideAnnotation"
	// OutsideSnapshot은 클러스터 외부에서 가져온 메트릭을 사용하여 SLI를 측정한다.
	OutsideSnapshot MeasurementMethod = "OutsideSnapshot"
)

// RunLocation은 측정이 실행되는 위치를 설명한다.
type RunLocation string

// RunTrigger는 측정 캡처를 유발하는 트리거를 설명한다.
type RunTrigger string

const (
	// RunLocationInside는 측정이 대상 환경 내부에서 실행됨을 나타낸다.
	RunLocationInside RunLocation = "inside"

	// RunLocationOutside는 측정이 대상 환경 외부에서 실행됨을 나타낸다.
	RunLocationOutside RunLocation = "outside"

	// RunTriggerNone은 특정 트리거가 사용되지 않음을 나타낸다.
	RunTriggerNone RunTrigger = "none"

	// RunTriggerAnnotation은 어노테이션에 의해 실행이 트리거됨을 나타낸다.
	RunTriggerAnnotation RunTrigger = "annotation"
)

// RunModeTyped holds the resolved run mode (typed version of RunMode).
// RunModeTyped는 해결된 실행 모드를 담습니다 (RunMode의 타입화된 버전).
type RunModeTyped struct {
	Location RunLocation
	Trigger  RunTrigger
}

// MapMethodToRunMode는 측정 방식을 프레임워크 소유의 실행 모드로 변환한다.
func MapMethodToRunMode(method MeasurementMethod) RunModeTyped {
	switch method {
	case InsideAnnotation:
		return RunModeTyped{Location: RunLocationInside, Trigger: RunTriggerAnnotation}
	case OutsideSnapshot:
		return RunModeTyped{Location: RunLocationOutside, Trigger: RunTriggerNone}
	case InsideSnapshot:
		fallthrough
	default:
		return RunModeTyped{Location: RunLocationInside, Trigger: RunTriggerNone}
	}
}
