package connectlog

import "strings"

// redactHeadersMap processes headers and redacts sensitive values
func redactHeadersMap(headers map[string][]string, redactHeaders []string) map[string][]string {
	redacted := make(map[string][]string, len(headers))
	for k, v := range headers {
		if shouldRedactHeader(k, redactHeaders) {
			redacted[k] = []string{"[REDACTED]"}
		} else {
			redacted[k] = v
		}
	}
	return redacted
}

// shouldRedactHeader determines if a header should be redacted
func shouldRedactHeader(key string, redactHeaders []string) bool {
	keyLower := strings.ToLower(key)
	for _, h := range redactHeaders {
		if strings.EqualFold(keyLower, h) {
			return true
		}
	}

	return keyLower == "authorization" ||
		strings.Contains(keyLower, "token") ||
		strings.Contains(keyLower, "secret") ||
		strings.Contains(keyLower, "password")
}
