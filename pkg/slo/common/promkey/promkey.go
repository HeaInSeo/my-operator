package promkey

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
)

// Parse parses a Prometheus metric key token into name + labels.
// token examples:
//
//	metric_name
//	metric_name{a="b",c="d"}
//
// It supports Prometheus label value escapes: \" \\ \n \t \r
// Parse는 Prometheus 메트릭 키 토큰을 이름 + 레이블로 파싱합니다.
// 토큰 예시:
//
//	metric_name
//	metric_name{a="b",c="d"}
//
// Prometheus 레이블 값 이스케이프를 지원합니다: \" \\ \n \t \r
func Parse(token string) (name string, labels map[string]string, err error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return "", nil, fmt.Errorf("empty token")
	}

	// No labels
	// 레이블 없음
	br := strings.IndexByte(token, '{')
	if br < 0 {
		return token, map[string]string{}, nil
	}

	// Must end with }
	// } 로 끝나야 함
	if !strings.HasSuffix(token, "}") {
		return "", nil, fmt.Errorf("invalid token (missing '}'): %q", token)
	}

	name = token[:br]
	inside := token[br+1 : len(token)-1]
	labels, err = parseLabels(inside)
	if err != nil {
		return "", nil, err
	}
	return name, labels, nil
}

// Format formats name + labels into canonical key string.
// Labels are sorted by key, values are escaped.
// Format은 이름 + 레이블을 정규 키 문자열로 포맷합니다.
// 레이블은 키로 정렬되며, 값은 이스케이프됩니다.
func Format(name string, labels map[string]string) string {
	if len(labels) == 0 {
		return name
	}
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	b.WriteString(name)
	b.WriteByte('{')
	for i, k := range keys {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteByte('"')
		b.WriteString(EscapeLabelValue(labels[k]))
		b.WriteByte('"')
	}
	b.WriteByte('}')
	return b.String()
}

// Canonicalize converts a raw token into canonical key string.
// Canonicalize는 원시 토큰을 정규 키 문자열로 변환합니다.
func Canonicalize(token string) (string, error) {
	name, labels, err := Parse(token)
	if err != nil {
		return "", err
	}
	return Format(name, labels), nil
}

func parseLabels(s string) (map[string]string, error) {
	labels := map[string]string{}
	i := 0
	for {
		// skip spaces/commas
		// 공백/쉼표 건너뛰기
		for i < len(s) && (s[i] == ' ' || s[i] == ',') {
			i++
		}
		if i >= len(s) {
			break
		}

		// parse key
		// 키 파싱
		start := i
		for i < len(s) && s[i] != '=' {
			i++
		}
		if i >= len(s) {
			return nil, fmt.Errorf("invalid labels (missing '='): %q", s)
		}
		key := strings.TrimSpace(s[start:i])
		i++ // '='

		// expect opening quote
		// 여는 따옴표 기대
		for i < len(s) && s[i] == ' ' {
			i++
		}
		if i >= len(s) || s[i] != '"' {
			return nil, fmt.Errorf("invalid labels (missing '\"' for %q): %q", key, s)
		}
		i++ // opening '"'

		// parse quoted value with escapes until closing quote
		// 닫는 따옴표까지 이스케이프가 포함된 인용 값 파싱
		var raw bytes.Buffer
		for {
			if i >= len(s) {
				return nil, fmt.Errorf("invalid labels (unterminated value for %q): %q", key, s)
			}
			ch := s[i]
			if ch == '"' {
				i++ // closing '"'
				break
			}
			if ch == '"' {
				i++ // 닫는 '"'
				break
			}
			if ch == '\\' {
				if i+1 >= len(s) {
					return nil, fmt.Errorf("invalid escape at end for %q: %q", key, s)
				}
				raw.WriteByte('\\')
				raw.WriteByte(s[i+1])
				i += 2
				continue
			}
			raw.WriteByte(ch)
			i++
		}

		val, err := UnescapeLabelValue(raw.String())
		if err != nil {
			return nil, fmt.Errorf("unescape label %q: %w", key, err)
		}
		labels[key] = val

		// trailing spaces handled by loop
		// 후행 공백은 루프에서 처리됨
	}
	return labels, nil
}

// EscapeLabelValue escapes a label value for Prometheus text format.
// EscapeLabelValue는 Prometheus 텍스트 형식을 위해 레이블 값을 이스케이프합니다.
func EscapeLabelValue(v string) string {
	var b strings.Builder
	for i := 0; i < len(v); i++ {
		switch v[i] {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		case '\n':
			b.WriteString(`\n`)
		case '\t':
			b.WriteString(`\t`)
		case '\r':
			b.WriteString(`\r`)
		default:
			b.WriteByte(v[i])
		}
	}
	return b.String()
}

// UnescapeLabelValue unescapes Prometheus label value escapes.
// UnescapeLabelValue는 Prometheus 레이블 값 이스케이프를 해제합니다.
func UnescapeLabelValue(v string) (string, error) {
	var b strings.Builder
	for i := 0; i < len(v); i++ {
		ch := v[i]
		if ch != '\\' {
			b.WriteByte(ch)
			continue
		}
		if i+1 >= len(v) {
			return "", fmt.Errorf("dangling escape")
		}
		i++
		switch v[i] {
		case '\\':
			b.WriteByte('\\')
		case '"':
			b.WriteByte('"')
		case 'n':
			b.WriteByte('\n')
		case 't':
			b.WriteByte('\t')
		case 'r':
			b.WriteByte('\r')
		default:
			// Prometheus generally treats unknown as literal char after backslash.
			// Prometheus는 일반적으로 백슬래시 뒤의 알 수 없는 문자를 리터럴 문자로 취급합니다.
			b.WriteByte(v[i])
		}
	}
	return b.String(), nil
}
