package harness

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/onsi/ginkgo/v2"
)

// Config defines the inputs for the SLO measurement session.
// type Config struct {
// 	Enabled bool	// false면 계측 훅 완전 스킵
// 	Namespace          string
// 	MetricsServiceName string
// 	TestCase           string
// 	Suite              string
// 	RunID              string
// 	ServiceAccountName string
// 	Token              string
// 	ArtifactsDir string
// 	Tags         map[string]string
// 	// 게측 코드에서 선택할 수 있도록 해줌.
// 	Method engine.Method
//     Fetcher fetch.MetricsFetcher
// }

// Attach registers BeforeEach/AfterEach hooks that call the provider function
// to get the config for the current test and manage the measurement session.
func Attach(provider func() SessionConfig) (*Session, error) {
	session := &Session{} // placeholder (impl is set in BeforeEach)

	// harness-level toggle (not exposed to operator's test code)
	enabled := isEnabledByEnv()

	ginkgo.BeforeEach(func() {
		if !enabled {
			session.reset(nil) // impl=nil so End() is a no-op/guarded
			return
		}

		cfg := provider()

		// Validate (keep your detailed messages)
		if strings.TrimSpace(cfg.Namespace) == "" {
			ginkgo.Fail(fmt.Sprintf(
				"harness: invalid config: Namespace is required (Suite=%q TestCase=%q RunID=%q MetricsServiceName=%q SA=%q ArtifactsDir=%q)",
				cfg.Suite, cfg.TestCase, cfg.RunID, cfg.MetricsServiceName, cfg.ServiceAccountName, cfg.ArtifactsDir,
			))
		}
		if strings.TrimSpace(cfg.MetricsServiceName) == "" {
			ginkgo.Fail(fmt.Sprintf(
				"harness: invalid config: MetricsServiceName is required (Namespace=%q Suite=%q TestCase=%q RunID=%q SA=%q)",
				cfg.Namespace, cfg.Suite, cfg.TestCase, cfg.RunID, cfg.ServiceAccountName,
			))
		}
		if strings.TrimSpace(cfg.Token) == "" {
			ginkgo.Fail(fmt.Sprintf(
				"harness: invalid config: Token is empty (Namespace=%q MetricsServiceName=%q Suite=%q TestCase=%q RunID=%q SA=%q)",
				cfg.Namespace, cfg.MetricsServiceName, cfg.Suite, cfg.TestCase, cfg.RunID, cfg.ServiceAccountName,
			))
		}

		// Auto-fill TestCase if empty
		if strings.TrimSpace(cfg.TestCase) == "" {
			cfg.TestCase = ginkgo.CurrentSpecReport().LeafNodeText
		}

		// Ensure Now is set (your Session.NewSession already does this, but safe here too)
		if cfg.Now == nil {
			cfg.Now = time.Now
		}

		newSess := NewSession(cfg)

		session.reset(newSess)
		session.Start()
	})

	ginkgo.AfterEach(func() {
		if !enabled {
			return
		}
		if session == nil || session.impl == nil {
			_, _ = fmt.Fprintf(ginkgo.GinkgoWriter, "SLO: session not initialized (enabled=true)\n")
			return
		}
		if _, err := session.End(context.Background()); err != nil {
			_, _ = fmt.Fprintf(ginkgo.GinkgoWriter, "SLO: End failed (skip): %v\n", err)
		}
	})

	return session, nil
}

func isEnabledByEnv() bool {
	// TODO: read from E2E_SLO_ENABLED or similar if needed.
	// For now, always enable since we are in the attach func
	return true
}
