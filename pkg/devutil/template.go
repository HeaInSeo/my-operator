package devutil

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

// RenderTemplateFile reads a template file at (rootDir + relPath), executes it with data,
// and returns the rendered bytes.
//
// - missing keys cause error (missingkey=error)
// - rootDir is typically the project root (e.g., repo root)
// RenderTemplateFile은 (rootDir + relPath)에 있는 템플릿 파일을 읽고, 데이터를 사용하여 실행한 뒤,
// 렌더링된 바이트를 반환합니다.
//
// - 키가 누락되면 에러가 발생합니다 (missingkey=error).
// - rootDir은 일반적으로 프로젝트 루트(예: repo root)입니다.
func RenderTemplateFile(rootDir, relPath string, data any) ([]byte, error) {
	if rootDir == "" {
		return nil, fmt.Errorf("rootDir is empty")
	}
	if relPath == "" {
		return nil, fmt.Errorf("relPath is empty")
	}

	path := filepath.Join(rootDir, relPath)

	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	tmpl, err := template.New(filepath.Base(relPath)).
		Option("missingkey=error").
		Parse(string(b))
	if err != nil {
		return nil, err
	}

	var out bytes.Buffer
	if err := tmpl.Execute(&out, data); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// RenderTemplateFileString renders a template file to a string.
// RenderTemplateFileString은 템플릿 파일을 렌더링하여 문자열로 반환합니다.
func RenderTemplateFileString(rootDir, relPath string, data any) (string, error) {
	b, err := RenderTemplateFile(rootDir, relPath, data)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
