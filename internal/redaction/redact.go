package redaction

import (
	"regexp"
	"strings"
)

var secretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)api[_-]?key\s*[:=]\s*[^\s]+`),
	regexp.MustCompile(`(?i)token\s*[:=]\s*[^\s]+`),
	regexp.MustCompile(`(?i)password\s*[:=]\s*[^\s]+`),
}

var sensitiveContextPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)password`),
	regexp.MustCompile(`(?i)mfa|2fa|one-time code|otp`),
	regexp.MustCompile(`(?i)credential|secret|token`),
	regexp.MustCompile(`(?i)bank|payment|billing|ssn|social security`),
	regexp.MustCompile(`(?i)health|medical|patient|phi`),
	regexp.MustCompile(`(?i)vault|keychain|password manager`),
}

func Text(input string) string {
	out := input
	for _, re := range secretPatterns {
		out = re.ReplaceAllString(out, "[REDACTED]")
	}
	return out
}

func URLAllowed(url string, allowlist, blocklist []string) bool {
	u := strings.ToLower(url)
	for _, denied := range blocklist {
		if denied != "" && strings.Contains(u, strings.ToLower(denied)) {
			return false
		}
	}
	if len(allowlist) == 0 {
		return true
	}
	for _, allowed := range allowlist {
		if allowed != "" && strings.Contains(u, strings.ToLower(allowed)) {
			return true
		}
	}
	return false
}

func SensitiveContext(values ...string) bool {
	for _, v := range values {
		raw := strings.TrimSpace(v)
		if raw == "" {
			continue
		}
		for _, p := range sensitiveContextPatterns {
			if p.MatchString(raw) {
				return true
			}
		}
	}
	return false
}
