````markdown
# 📄 팀 코드 작성 규칙 v1.0 (Go / Kubernetes Operator 전용)
**부제: Cloud-Native & Library-First**

## 목적
우리는 오퍼레이터 코드를 다음 기준으로 유지한다.

- **Readable**: 읽기 쉽다
- **Reusable**: 재사용 가능하다
- **Upgradeable**: 업그레이드(의존성/버전 변화)에 강하다
- **Testable**: 테스트하기 쉽다

> 원칙: **Reconcile는 슬림하게, 라이브러리는 순수하게, 의존성은 격리한다.**

---

## 목차
1. [코드 승격(Promotion) 전략: Reconcile 슬림화의 핵심](#1-코드-승격promotion-전략-reconcile-슬림화의-핵심)
2. [의존성 및 순수성(Purity) 규칙](#2-의존성-및-순수성purity-규칙)
3. [계측(Instrumentation) 규칙: 불침투 철학 고정](#3-계측instrumentation-규칙-불침투-철학-고정)
4. [성능 정책: 측정 없이는 최적화도 없다](#4-성능-정책-측정-없이는-최적화도-없다)
5. [PR 리뷰 체크리스트](#5-pr-리뷰-체크리스트)
6. [실전 예시: 승격이 필요한 “결정적 장면” 2개](#6-실전-예시-승격이-필요한-결정적-장면-2개)
7. [AI/CI Enforcement: .antigravity_rules & golangci-lint](#7-aici-enforcement-antigravity_rules--golangci-lint)

---

## 1. 코드 승격(Promotion) 전략: Reconcile 슬림화의 핵심

오퍼레이터의 `Reconcile()`은 비대해지기 쉽다.  
우리는 로직의 **중요도/재사용성/의존성 위험도**에 따라 단계를 **승격(Promotion)** 한다.

### 1.1 승격 레벨(P0~P5)

| 레벨 | 명칭 | 위치(Go 관례) | 승격 기준(Trigger) |
|---|---|---|---|
| **P0** | Inline | `Reconcile()` 내부 | 5줄 이하, 단순 가드/분기 |
| **P1** | Closure | 함수 내부 클로저 | 콜백이 본질인 곳에서만, 짧고 캡처 적음 |
| **P2** | File-private | 같은 파일 소문자 함수 | 15줄↑, 분기 2개↑, 테스트 필요 |
| **P3** | Package-private | 같은 패키지 소문자 유틸 | 여러 컨트롤러 공통 패턴(파이널라이저/컨디션/리큐 등) |
| **P4** | Internal Component | `internal/` 또는 `test/e2e/` 접점 레이어 | **K8s/controller-runtime/외부 SDK 의존성 격리** 필요 |
| **P5** | Public Library | `pkg/` (외부 import 가능) | 타 프로젝트에서도 재사용 가능한 “순수 라이브러리” |

### 1.2 P4 vs P5 경계(필수 정의)

- **P5 (`pkg/`)**: Public API에 `k8s.io/*`, `sigs.k8s.io/controller-runtime/*` 타입이 **절대 등장하지 않는다**  
  - 허용: `context`, `time`, `io`, `encoding/json` 등 **표준 라이브러리 중심**
  - 목표: **순수 계산 / 파서 / 요약 스키마 / 엔진(비K8s)**

- **P4 (`internal/`, `test/e2e/`)**: K8s 접근(클라이언트/SSA/리소스 렌더/inside-curl 등)과 환경 의존 로직을 모아두는 **접점 레이어**
  - 목표: **K8s 버전/라이브러리 변화의 충격을 흡수**

### 1.3 V4 전환 규칙 (프로젝트 기준선)

- `pkg/slo`는 **P5로 고정**: 순수 계산/파서/요약 스키마/엔진만 남긴다.
- K8s 의존(inside snapshot, curl pod, kubectl, controller-runtime)은 **P4로 이동**: `internal/...` 또는 `test/e2e/...`로 내린다.

---

## 2. 의존성 및 순수성(Purity) 규칙

외부 변화에 흔들리지 않는 코드를 만들기 위한 철칙이다.

### 2.1 Dependency Isolation (의존성 격리)

#### (1) Public API 누수 금지 (P5 필수)
- `pkg/*` 패키지의 함수 인자/리턴/필드에 `k8s.io/*`, `controller-runtime` 타입이 등장하면 안 된다.
- 외부 타입은 **adapter(P4)** 에서만 다루고, P5로 들어올 때는 **도메인 타입/원시 타입**으로 변환한다.

#### (2) Adapter 패턴 강제 (P4 표준)
- SSA/Patch/Discovery/RESTMapper 등 복잡한 K8s API는 `internal/adapter`에 감싼다.
- 컨트롤러는 adapter의 **인터페이스만** 호출한다.

### 2.2 Blank Import (`_ "pkg"`)
- **허용 위치**: `main.go` 또는 `cmd/*` 엔트리포인트 한정
- **의무 문서화**: 주석으로 “어떤 사이드이펙트(등록/초기화)를 기대하는지” 명시
- **금지**: `pkg/*` (P5) 라이브러리 코드에서 blank import

---

## 3. 계측(Instrumentation) 규칙: 불침투 철학 고정

> 우리의 계측 철학은 “프로덕션 오퍼레이터 코드에 계측을 삽입하지 않는다(불침투)”로 고정한다.

1) **불침투(Non-invasive)**
- 오퍼레이터 프로덕션 `Reconcile()` 경로에 계측 코드를 삽입하지 않는다.

2) **계측 위치 고정**
- 계측은 **E2E 훅(테스트 시작/종료)** 에서만 수행한다.
- 계측 실패는 테스트 실패가 아니다(**경고 + skip**).

3) **산출물 고정**
- 원시 소스: `/metrics`
- 결과 요약: `sli-summary.json`

> 두 개 외 형식을 추가하려면 팀 합의가 필요하다.

---

## 4. 성능 정책: “측정 없이는 최적화도 없다”

- 기본: **가독성이 성능보다 우선**한다.
- 예외: 벤치마크(`go test -bench`) 또는 프로파일(pprof) 근거가 PR에 포함될 때만 최적화 허용
- 최적화는 반드시 아래를 기록한다:
  - 왜 필요한지
  - 단순한 대안이 왜 실패했는지
  - 벤치/프로파일 근거(커맨드/결과 요약)

---

## 5. PR 리뷰 체크리스트

리뷰어는 다음 질문을 던진다.

- **재사용성**: 이 로직은 P2/P3로 승격해야 하는가?
- **의존성**: P5에서 K8s/controller-runtime 타입이 새고 있지 않은가?
- **가독성**: 클로저(P1)가 컨텍스트를 과도 캡처해 “암호문”이 되었는가?
- **계측**: 계측이 Reconcile 경로로 들어오지 않았는가?
- **성능**: 최적화가 있다면 근거가 포함되어 있는가?

---

## 6. 실전 예시: 승격이 필요한 “결정적 장면” 2개

### Case 1) P1 → P2: Reconcile 내부 클로저를 파일 헬퍼로
**목표**: Reconcile 흐름을 살리고, 단위 테스트 가능하게 만들기

#### Before (P1: 클로저가 길어지고 캡처 많음)

```go
healthOK := func(obj *appsv1.Deployment) bool {
    if obj.Status.Replicas == 0 {
        return false
    }
    if obj.Status.ReadyReplicas < obj.Status.Replicas {
        return false
    }
    // ... 점점 늘어나 20줄+
    return true
}

if !healthOK(dep) {
    return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
}
````

#### After (P2: file-private helper)

```go
if !isDeploymentHealthy(dep) {
    return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
}

func isDeploymentHealthy(dep *appsv1.Deployment) bool {
    if dep.Status.Replicas == 0 {
        return false
    }
    if dep.Status.ReadyReplicas < dep.Status.Replicas {
        return false
    }
    return true
}
```

---

### Case 2) P4 ↔ P5 경계: K8s 의존은 internal로, 순수 계산은 pkg로

**목표**: `pkg/slo`의 Public API에서 K8s 의존을 “0”으로 만들기

#### P4 (internal): K8s/환경 의존 “접점 레이어”

```go
// internal/fetcher/metrics_source.go
package fetcher

import "context"

// K8s 접근(inside-curl, kubectl, client) 등을 통해 raw 텍스트를 가져오는 책임
type MetricsSource interface {
    Fetch(ctx context.Context) (string, error) // Prometheus text raw
}
```

#### P5 (pkg): 순수 파싱/계산/요약

```go
// pkg/slo/metrics.go
package slo

type MetricMap map[string]float64

func ParsePromText(raw string) (MetricMap, error) {
    // 순수 파싱: 입력(raw) -> 출력(map) / 외부 의존 없음
    // ...
    return MetricMap{}, nil
}

func ComputeDelta(start, end MetricMap) MetricMap {
    out := MetricMap{}
    for k, vEnd := range end {
        out[k] = vEnd - start[k]
    }
    return out
}
```

---

## 7. AI/CI Enforcement: .antigravity_rules & golangci-lint  

### 7.1 .antigravity_rules (AI 자동 감지 규약)

#### Promotion Detection (승격 감지)

* `Reconcile` 함수 내 **연속 15줄 이상의 인라인 로직 블록** 발견 시:

  * “P2(File-private helper)로 승격을 검토하세요”
* 로컬 클로저가 **외부 변수 캡처 2개 이상**이면:

  * “P2로 분리하고 시그니처를 명확히 하세요”
* `Reconcile()`에서 `Status().Update()` 또는 `Patch()`가 **2회 이상 등장**하면:

  * “상태/적용 로직을 P3/P4로 승격 검토”

#### Dependency Leak (의존성 누수 방지)

* `pkg/slo`(P5) exported API에서 아래 타입/패키지가 등장하면 경고:

  * `k8s.io/*`
  * `sigs.k8s.io/controller-runtime/*`
* 메시지:

  * “Public API에 K8s 의존이 누출되었습니다. internal/adapter(P4)로 이동 후 도메인 타입으로 변환하세요.”

#### Blank Import Audit

* `_ "..."`가 `main.go` 또는 `cmd/*` 밖에서 발견되면 즉시 경고
* 가능하면 명시적 등록(Explicit registration) 대안을 제시

---

### 7.2 golangci-lint (시스템적으로 규칙을 강제)

권장 조합:

* **depguard**: `pkg/slo`에서 `k8s.io/*`, `controller-runtime` import 금지
* **cyclop** 또는 **gocognit**: `Reconcile()` 복잡도 상한 설정 → 승격 유도
* **revive**: 스타일/가독성 규칙 보조
* **errcheck**: 에러 무시 방지

> 목표: “좋은 습관”이 아니라 “CI에서 자동으로 지켜지는 규칙”이 되게 한다.



