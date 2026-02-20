package e2eutil

import (
	"fmt"

	"github.com/onsi/ginkgo/v2"

	"github.com/yeongki/my-operator/pkg/slo"
)

// 사용 예시
// logger := slo.NewLogger(utils.GinkgoLog) // nil이면 noop
// logger.Logf("hello %s", "world")

// GinkgoLogger adapts slo.Logger to GinkgoWriter.
// GinkgoLogger는 slo.Logger를 GinkgoWriter에 맞게 어댑터합니다.
type GinkgoLogger struct{}

// Logf logs a formatted string.
// Logf는 포맷된 문자열을 로그로 남깁니다.
func (GinkgoLogger) Logf(format string, args ...any) {
	_, _ = fmt.Fprintf(ginkgo.GinkgoWriter, format+"\n", args...)
}

// Compile-time check
// 컴파일 타임 검사
var _ slo.Logger = (*GinkgoLogger)(nil)

// GinkgoLog is a ready-to-use instance only for test files.
// Ready-to-use instance
// GinkgoLog는 테스트 파일 전용의 즉시 사용 가능한 인스턴스입니다.
// 사용 준비된 인스턴스
var GinkgoLog slo.Logger = GinkgoLogger{}
