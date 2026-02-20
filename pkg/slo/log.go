package slo

// Logger는 pkg/slo를 위한 최소한의 로깅 계약
type Logger interface {
	Logf(format string, args ...any)
}

type nopLogger struct{}

// Logf는 Logger 인터페이스를 구현한다.
func (nopLogger) Logf(string, ...any) {}

// NewLogger는 안전한 Logger를 반환합니다. l이 nil이면 no-op을 반환한다.
func NewLogger(l Logger) Logger {
	if l == nil {
		return nopLogger{}
	}
	return l
}

// 원한다면 사용할 수 있는 내보내진 NopLogger 싱글톤
var NopLogger Logger = nopLogger{}
