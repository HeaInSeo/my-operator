package harness

import "strings"

// SanitizeFilename makes a string safe-ish for filenames.
// TODO(shared): Consider moving this helper to pkg/slo/common (or devutil) so both harness and other packages reuse it.
//
//	If specs/config are loaded from files, this will be used across packages for artifact naming and should be stable.
func SanitizeFilename(s string) string {
	t := strings.TrimSpace(s)
	if t == "" {
		return "unknown"
	}
	repl := strings.NewReplacer(
		"/", "_", "\\", "_", " ", "_", ":", "_", ";", "_",
		"\"", "_", "'", "_", "\n", "_", "\r", "_", "\t", "_",
	)
	t = repl.Replace(t)
	// keep it short-ish
	if len(t) > 120 {
		t = t[:120]
	}
	return t
}
