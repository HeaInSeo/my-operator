package fetch

import (
	"context"
	"time"
)

// Sample is one snapshot at a point in time.
// Sample은 특정 시점의 스냅샷 하나입니다.
type Sample struct {
	At     time.Time
	Values map[string]float64 // metricKey -> value
}

// MetricsFetcher fetches one snapshot. Implementations decide how to obtain it.
// - outside: curl /metrics (via Pod, port-forward, HTTP)
// - inside: direct HTTP to localhost
// - trigger: could fetch from /metrics or status/log-derived metrics
// MetricsFetcher는 하나의 스냅샷을 가져옵니다. 구현체에서 가져오는 방법을 결정합니다.
// - outside: curl /metrics (Pod, port-forward, HTTP를 통해)
// - inside: localhost로 직접 HTTP 요청
// - trigger: /metrics 또는 status/log 파생 메트릭에서 가져올 수 있음
type MetricsFetcher interface {
	Fetch(ctx context.Context, at time.Time) (Sample, error)
}
