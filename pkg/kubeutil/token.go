package kubeutil

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/yeongki/my-operator/pkg/slo"
)

type tokenRequest struct {
	Status struct {
		Token string `json:"token"`
	} `json:"status"`
}

const tokenRequestBody = `{"apiVersion":"authentication.k8s.io/v1","kind":"TokenRequest"}`

// TODO(kubeutil): When we add TokenRequest options (audiences/expirationSeconds/etc),
// stop using a raw JSON string and marshal a struct instead.
// Rationale: easier to extend safely (optional fields), avoids fragile string edits,
// and produces correct JSON consistently.

// ServiceAccountToken requests a token for the given ServiceAccount.
// - Retries until ctx is done.
// - logger may be nil (no-op).
func ServiceAccountToken(ctx context.Context, logger slo.Logger, r CmdRunner, ns, sa string) (string, error) {
	logger = slo.NewLogger(logger)
	if r == nil {
		r = DefaultRunner{}
	}

	if err := ctx.Err(); err != nil {
		return "", err
	}

	var lastErr error
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	tryOnce := func() (string, error) {
		cmd := exec.Command("kubectl", "create", "--raw",
			fmt.Sprintf("/api/v1/namespaces/%s/serviceaccounts/%s/token", ns, sa),
			"-f", "-",
		)
		cmd.Stdin = strings.NewReader(tokenRequestBody)

		stdout, err := r.Run(ctx, logger, cmd) // ✅ ctx 반영
		if err != nil {
			return "", fmt.Errorf("token request failed (ns=%s sa=%s): %w", ns, sa, err)
		}

		var tr tokenRequest
		if err := json.Unmarshal([]byte(stdout), &tr); err != nil {
			return "", fmt.Errorf("token response json parse failed: %w (body=%q)", err, stdout)
		}
		if tr.Status.Token == "" {
			return "", fmt.Errorf("token is empty")
		}
		return tr.Status.Token, nil
	}

	if tok, err := tryOnce(); err == nil {
		return tok, nil
	} else {
		lastErr = err
		logger.Logf("token not ready yet: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			if lastErr == nil {
				lastErr = ctx.Err()
			}
			return "", lastErr
		case <-ticker.C:
			tok, err := tryOnce()
			if err == nil {
				return tok, nil
			}
			lastErr = err
			logger.Logf("token not ready yet: %v", err)
		}
	}
}
