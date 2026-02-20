package harness

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yeongki/my-operator/pkg/slo/fetch"
)

func TestNewSession(t *testing.T) {
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	cfg := SessionConfig{
		Namespace:          "test-ns",
		MetricsServiceName: "test-svc",
		TestCase:           "TestCase A",
		Suite:              "e2e",
		RunID:              "run-123",
		Tags:               map[string]string{"env": "ci"},
		Now:                func() time.Time { return now },
	}

	sess := NewSession(cfg)

	assert.Equal(t, "test-ns", sess.impl.Config.Namespace)
	assert.Equal(t, "run-123", sess.impl.RunID)
	assert.Equal(t, "e2e", sess.impl.Tags["suite"])
	assert.Equal(t, "TestCase A", sess.impl.Tags["test_case"])
	assert.Equal(t, "ci", sess.impl.Tags["env"])
	assert.NotNil(t, sess.impl.writer)
}

func TestSession_AutoRunID(t *testing.T) {
	now := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	cfg := SessionConfig{
		Namespace: "ns",
		Now:       func() time.Time { return now },
	}

	sess := NewSession(cfg)
	assert.Equal(t, "local-1704103200", sess.impl.RunID)
}

func TestSession_End(t *testing.T) {
	// Mock fetcher
	mockFetcher := &mockFetcher{}

	cfg := SessionConfig{
		Namespace: "ns",
		TestCase:  "test",
		Fetcher:   mockFetcher,
	}
	sess := NewSession(cfg)
	sess.Start()

	summary, err := sess.End(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, summary)
	assert.Equal(t, "v4", summary.Config.Format)
}

type mockFetcher struct{}

func (m *mockFetcher) Fetch(ctx context.Context, at time.Time) (fetch.Sample, error) {
	return fetch.Sample{
		At:     at,
		Values: map[string]float64{"foo": 1.0},
	}, nil
}
