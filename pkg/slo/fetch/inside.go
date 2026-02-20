package fetch

import "context"

// InsideSnapshotFetch fetches metrics using the inside CurlPod-only boundary.
// InsideSnapshotFetch는 내부 CurlPod 전용 경계를 사용하여 메트릭을 가져옵니다.
func InsideSnapshotFetch(ctx context.Context, fetchFunc func(context.Context) (string, error)) (string, []string) {
	body, err := fetchFunc(ctx)
	if err != nil {
		return "", []string{err.Error()}
	}
	return body, nil
}
