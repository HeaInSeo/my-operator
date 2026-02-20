package devutil

import "time"

// TestStartTimeAnnoKey is the annotation key for test start time.
// TestStartTimeAnnoKey는 테스트 시작 시간에 대한 어노테이션 키입니다.
const TestStartTimeAnnoKey = "test/start-time"

// SetTestStartTimeAnno sets test/start-time annotation to current UTC time (RFC3339Nano).
// Glue-layer helper: keeps core independent from k8s types (metav1.Object etc.).
// SetTestStartTimeAnno는 테스트 시작 시간 어노테이션을 현재 UTC 시간(RFC3339Nano)으로 설정합니다.
// Glue-layer 헬퍼: 코어를 k8s 타입(metav1.Object 등)으로부터 독립적으로 유지합니다.
func SetTestStartTimeAnno(ann map[string]string) map[string]string {
	return SetTestStartTimeAnnoAt(ann, time.Now())
}

// SetTestStartTimeAnnoAt sets test/start-time annotation using the provided time.
// Prefer this in callers that want one captured "now" reused across multiple objects.
// SetTestStartTimeAnnoAt은 제공된 시간을 사용하여 테스트 시작 시간 어노테이션을 설정합니다.
// 여러 객체에 걸쳐 캡처된 하나의 "지금(now)"을 재사용하려는 호출자에게 권장됩니다.
func SetTestStartTimeAnnoAt(ann map[string]string, now time.Time) map[string]string {
	if ann == nil {
		ann = map[string]string{}
	}
	ann[TestStartTimeAnnoKey] = now.UTC().Format(time.RFC3339Nano)
	return ann
}
