package curlpod

import (
	"context"
	"time"
)

// CurlPod encapsulates the curl pod lifecycle without external adapters.
// CurlPod는 외부 어댑터 없이 curl 파드 수명 주기를 캡슐화합니다.
type CurlPod struct {
	Client             *Client
	Namespace          string
	MetricsServiceName string
	ServiceAccountName string
	Token              string

	Image            string
	ServiceURLFormat string
}

// Run executes the curl pod lifecycle and returns logs.
// Run은 curl 파드 수명 주기를 실행하고 로그를 반환합니다.
func (c *CurlPod) Run(ctx context.Context, waitTimeout time.Duration, logsTimeout time.Duration) (string, error) {
	client := c.Client
	if client == nil {
		client = New(nil, nil)
	}
	if c.Image != "" {
		client.Image = c.Image
	}
	if c.ServiceURLFormat != "" {
		client.ServiceURLFormat = c.ServiceURLFormat
	}

	podName, err := client.RunOnce(ctx, c.Namespace, c.Token, c.MetricsServiceName, c.ServiceAccountName)
	if err != nil {
		return "", err
	}

	waitCtx, waitCancel := context.WithTimeout(ctx, waitTimeout)
	defer waitCancel()
	if err := client.WaitDone(waitCtx, c.Namespace, podName, 2*time.Second); err != nil {
		_ = client.DeletePodNoWait(ctx, c.Namespace, podName)
		return "", err
	}

	logCtx, logCancel := context.WithTimeout(ctx, logsTimeout)
	defer logCancel()
	out, err := client.Logs(logCtx, c.Namespace, podName)
	_ = client.DeletePodNoWait(ctx, c.Namespace, podName)
	return out, err
}
