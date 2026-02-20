package harness

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/yeongki/my-operator/pkg/slo/engine"
	"github.com/yeongki/my-operator/pkg/slo/fetch"
	"github.com/yeongki/my-operator/pkg/slo/fetch/curlpod"
	"github.com/yeongki/my-operator/pkg/slo/fetch/promtext"
	"github.com/yeongki/my-operator/pkg/slo/spec"
	"github.com/yeongki/my-operator/pkg/slo/summary"
	"github.com/yeongki/my-operator/pkg/slo/tags"
)

// SessionConfig contains session inputs and defaults.
// SessionConfig는 세션 입력값과 기본값을 포함함.
type SessionConfig struct {
	Namespace          string
	MetricsServiceName string
	TestCase           string
	Suite              string

	// Optional inputs
	RunID string
	Tags  map[string]string

	ServiceAccountName string
	Token              string
	ArtifactsDir       string

	Method engine.MeasurementMethod
	Now    func() time.Time

	// Optional overrides
	Specs   []spec.SLISpec
	Fetcher fetch.MetricsFetcher
	Writer  summary.Writer
}

type sessionImpl struct {
	Config SessionConfig

	// Tunables (defaults are set in NewSession)
	ServiceURLFormat string
	// TODO: 향후 추가되거나 올릴예정임.
	CurlImage string
	// ServiceURLFormat 에서 결정됨. 일단 주석으로 남겨둠, 혹시 필요하면 살림.
	//MetricsPort      int
	// 추후
	// type SessionConfig struct {
	//     MetricsScheme string // "https"
	//     MetricsPort   int    // 8443
	//     MetricsPath   string // "/metrics"
	// } 이런식으로 필요할지 생각해보자. 일단 주석으로 남겨둠.

	ScrapeTimeout      time.Duration
	WaitPodDoneTimeout time.Duration
	LogsTimeout        time.Duration

	// Normalized (derived) runtime values
	RunID string
	Tags  map[string]string

	Warnings []string

	specs   []spec.SLISpec
	fetcher fetch.MetricsFetcher
	writer  summary.Writer

	started time.Time
}

type Session struct {
	impl *sessionImpl
}

// NewSession builds a session with defaults applied.
// NewSession은 기본값이 적용된 세션을 생성함.
func NewSession(cfg SessionConfig) *Session {
	// Defaults
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	// TODO: 기본적으로 InsideSnapshot을 사용한다. 그런데 오류처리할지 고민중
	if cfg.Method == "" {
		cfg.Method = engine.InsideSnapshot
	}

	runID := strings.TrimSpace(cfg.RunID)
	if runID == "" {
		runID = fmt.Sprintf("local-%d", cfg.Now().Unix())
	}

	autoTags := tags.AutoTags(tags.AutoTagsInput{
		Suite:     cfg.Suite,
		TestCase:  cfg.TestCase,
		Namespace: cfg.Namespace,
		RunID:     runID,
	})

	mergedTags := tags.MergeTags(cfg.Tags, autoTags)

	// TODO:Specs default, 이건 확인해봐야 한다.
	resolvedSpecs := defaultSpecs(cfg.Specs)

	// Writer default
	w := cfg.Writer
	if w == nil {
		w = summary.NewJSONFileWriter()
	}

	impl := &sessionImpl{
		Config: cfg,

		//MetricsPort:      8443,
		ServiceURLFormat: "https://%s.%s.svc:8443/metrics",
		CurlImage:        "curlimages/curl:latest",

		ScrapeTimeout:      2 * time.Minute,
		WaitPodDoneTimeout: 5 * time.Minute,
		LogsTimeout:        2 * time.Minute,

		RunID: runID,
		Tags:  mergedTags,

		specs:   resolvedSpecs,
		fetcher: cfg.Fetcher,
		writer:  w,
	}

	return &Session{impl: impl}
}

// reset replaces internal runtime state WITHOUT copying the whole Session struct.
// reset은 전체 Session 구조체를 복사하지 않고 내부 런타임 상태를 교체함.
// NOTE: This is NOT a deep copy. from must not be used after calling reset.
func (s *Session) reset(from *Session) {
	if s == nil {
		return
	}

	if from == nil {
		s.impl = nil
		return
	}

	s.impl = from.impl
}

// ShouldWriteArtifacts reports whether session should write summary output.
// ShouldWriteArtifacts는 세션이 요약 출력을 기록해야 하는지 여부를 보고함.
func (s *Session) ShouldWriteArtifacts() bool {
	// 여기서, s != nil 체크를 하는 이유는 Ginkgo 훅 + placeholder 패턴 때문에, 사용될 수 있기때문에 panic 방지용으로 넣음.
	return s != nil && s.impl != nil && s.impl.Config.ArtifactsDir != ""
}

// NextSummaryPath returns a unique summary path by appending -<n> on collisions.
// NextSummaryPath는 충돌 시 -<n>을 추가하여 고유한 요약 경로를 반환함.
func (s *Session) NextSummaryPath(filename string) (string, error) {
	// s == nil 체크는 일부러 하는 것이다. 일반적으로는 필요없지만 지금 케이스에서는 필요하다.
	if s == nil || s.impl == nil {
		return "", fmt.Errorf("harness: session not initialized")
	}

	if strings.TrimSpace(s.impl.Config.ArtifactsDir) == "" {
		return "", nil
	}

	base := filepath.Join(s.impl.Config.ArtifactsDir, filename)
	path := base
	for i := 1; ; i++ {
		_, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				return path, nil
			}
			return "", err
		}
		path = fmt.Sprintf("%s-%d", base, i)
	}
}

// AddWarning records a warning message for BestEffort mode.
// AddWarning은 BestEffort 모드에 대한 경고 메시지를 기록함.
func (s *Session) AddWarning(message string) {
	if s == nil || s.impl == nil || strings.TrimSpace(message) == "" {
		return
	}
	s.impl.Warnings = append(s.impl.Warnings, message)
}

// Start begins measurement.
// Start는 측정을 시작함.
func (s *Session) Start() {
	if s == nil || s.impl == nil {
		return
	}
	// Defensive: handle nil/invalid Now() injection.
	now := s.impl.Config.Now
	if now == nil {
		now = time.Now
	}

	t := now()
	if t.IsZero() {
		t = time.Now()
	}
	s.impl.started = t
}

// End는 측정을 완료함.
// TODO(harness-thin):
//  - harness는 "연결자(glue)"로 얇아야 한다. 현재는 curlpod/promtext 구현까지 포함되어 두텁다.
//  - 목표: Method/FetchStrategy(또는 Source) 선택은 e2e_test.go(provider)가 결정하고,
//    harness는 검증/시간윈도우(StartedAt/FinishedAt)/Engine 실행 + artifact 경로 계산만 담당한다.
//  - 조치:
//    1) newCurlPodFetcher/curlPodFetcher/parsePrometheusText를 harness 밖(예: harness/source/curlpod 또는 test-side adapter)으로 이동.
//    2) End()에서 "fetcher nil이면 curlpod" 같은 암묵 기본값 제거(또는 NewSession 단계에서만 기본값 처리).
//    3) Method는 Config.Method를 그대로 사용(하드코딩 금지)하고, 장기적으로는 Method도 필수 입력으로 강제.
//    4) (선택) Source.Build(ctx) -> fetcher 생성 패턴 도입하여, K8s 의존 구현은 adapter에 격리.

func (s *Session) End(ctx context.Context) (*summary.Summary, error) {
	if s == nil || s.impl == nil {
		return nil, fmt.Errorf("harness: session not initialized")
	}

	if ctx == nil {
		ctx = context.Background()
	}

	started := s.impl.started
	if started.IsZero() {
		return nil, fmt.Errorf("harness: Start() was not called (RunID=%q TestCase=%q)", s.impl.RunID, s.impl.Config.TestCase)

	}

	// Important: time synchronization and injection time should be discussed separately and documented.
	now := s.impl.Config.Now
	if now == nil {
		now = time.Now
	}
	finished := now()

	// TODO(harness-thin):
	//  - harness는 "연결자(glue)"로 얇아야 한다. 현재는 curlpod/promtext 구현까지 포함되어 두텁다.
	//  - 목표: Method/FetchStrategy(또는 Source) 선택은 e2e_test.go(provider)가 결정하고,
	//    harness는 검증/시간윈도우(StartedAt/FinishedAt)/Engine 실행 + artifact 경로 계산만 담당한다.
	//  - 조치:
	//    1) newCurlPodFetcher/curlPodFetcher/parsePrometheusText를 harness 밖으로 이동.
	//    2) End()의 fetcher fallback 정책 정리(장기적으로는 명시 주입).
	//    3) Method 하드코딩 금지, 장기적으로 Method 필수화.
	m := s.impl.Config.Method
	if m == "" {
		m = engine.InsideSnapshot
	}

	fetcher := s.impl.fetcher
	if fetcher == nil {
		fetcher = newCurlPodFetcher(s.impl)
	}

	eng := engine.New(fetcher, s.impl.writer, nil)

	outPath := ""
	if s.ShouldWriteArtifacts() {
		filename := fmt.Sprintf(
			"sli-summary.%s.%s.json",
			SanitizeFilename(s.impl.RunID),
			SanitizeFilename(s.impl.Config.TestCase),
		)
		path, err := s.NextSummaryPath(filename)
		if err != nil {
			return nil, err
		}
		outPath = path
	}

	return engine.ExecuteStandard(ctx, eng, engine.ExecuteRequestStandard{
		Method: m,
		Config: engine.RunConfig{
			RunID:      s.impl.RunID,
			StartedAt:  started,
			FinishedAt: finished,
			Format:     "v4",
			Tags:       s.impl.Tags,
		},
		Specs:   s.impl.specs,
		OutPath: outPath,
	})
}

// ---- (TEMP) Default Fetcher: curlpod + promtext ----
// TODO(harness-thin): 아래 구현은 pkg/slo/fetch/* 또는 test-side adapter로 옮기는 게 목표.

type curlPodFetcher struct {
	impl *sessionImpl
	pod  *curlpod.CurlPod
}

func newCurlPodFetcher(impl *sessionImpl) fetch.MetricsFetcher {
	return &curlPodFetcher{
		impl: impl,
		pod: &curlpod.CurlPod{
			// NOTE: CurlPod.Client는 nil이면 내부에서 New(nil,nil)로 default 생성됨.
			Namespace:          impl.Config.Namespace,
			MetricsServiceName: impl.Config.MetricsServiceName,
			ServiceAccountName: impl.Config.ServiceAccountName,
			Token:              impl.Config.Token,
			Image:              impl.CurlImage,
			ServiceURLFormat:   impl.ServiceURLFormat,
		},
	}
}

// Fetch retrieves a metric sample.
// Fetch는 메트릭 샘플을 조회함.
func (f *curlPodFetcher) Fetch(ctx context.Context, at time.Time) (fetch.Sample, error) {
	podCtx, cancel := context.WithTimeout(ctx, f.impl.ScrapeTimeout)
	defer cancel()

	raw, err := f.pod.Run(podCtx, f.impl.WaitPodDoneTimeout, f.impl.LogsTimeout)
	if err != nil {
		return fetch.Sample{}, err
	}

	values, err := parsePrometheusText(raw)
	if err != nil {
		return fetch.Sample{}, err
	}

	return fetch.Sample{
		At:     at,
		Values: values,
	}, nil
}

func parsePrometheusText(raw string) (map[string]float64, error) {
	base, err := promtext.ParseTextToMap(strings.NewReader(raw))
	if err != nil {
		return nil, err
	}

	out := map[string]float64{}
	for key, val := range base {
		out[key] = val
		if idx := strings.Index(key, "{"); idx > 0 {
			name := key[:idx]
			out[name] = out[name] + val
		}
	}
	return out, nil
}

func defaultSpecs(specs []spec.SLISpec) []spec.SLISpec {
	if specs != nil {
		return specs
	}
	// TODO: Replace DefaultV3Specs with a cleaner 'DefaultSpecs' or load from file.
	return DefaultV3Specs()
}
