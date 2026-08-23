package leonardo

import (
	"fmt"
	"strconv"
	"strings"
)

// normalizeCookieInput accepts a Cookie header, a single session token, or a
// Netscape HTTP Cookie File and returns one safe, single-line Cookie header.
func normalizeCookieInput(raw string) (string, error) {
	value := strings.TrimSpace(strings.TrimPrefix(raw, "\ufeff"))
	if value == "" {
		return "", nil
	}
	if looksLikeNetscapeCookieFile(value) {
		return parseNetscapeCookieFile(value)
	}
	if strings.HasPrefix(strings.ToLower(value), "cookie:") {
		value = strings.TrimSpace(value[len("cookie:"):])
	}
	if strings.ContainsAny(value, "\r\n\x00") {
		return "", fmt.Errorf("cookie contains a newline or control character")
	}
	if !strings.Contains(value, "=") {
		return "__Secure-better-auth.session_token=" + value, nil
	}
	if err := validateCookieHeader(value); err != nil {
		return "", err
	}
	return value, nil
}

func looksLikeNetscapeCookieFile(value string) bool {
	for _, rawLine := range strings.Split(value, "\n") {
		line := strings.TrimSpace(strings.TrimSuffix(rawLine, "\r"))
		lower := strings.ToLower(line)
		if strings.HasPrefix(lower, "# netscape http cookie file") || strings.HasPrefix(lower, "#httponly_") {
			return true
		}
		if line != "" && !strings.HasPrefix(line, "#") && strings.Contains(line, "\t") {
			return true
		}
	}
	return false
}

func parseNetscapeCookieFile(value string) (string, error) {
	parts := make([]string, 0, 16)
	positions := make(map[string]int)
	sawRecord := false

	for lineNumber, rawLine := range strings.Split(value, "\n") {
		line := strings.TrimSuffix(rawLine, "\r")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		lower := strings.ToLower(trimmed)
		if strings.HasPrefix(lower, "#httponly_") {
			line = trimmed[len("#httponly_"):]
		} else if strings.HasPrefix(trimmed, "#") {
			continue
		}

		fields := strings.SplitN(line, "\t", 7)
		if len(fields) != 7 {
			return "", fmt.Errorf("invalid Netscape cookie record on line %d: expected 7 tab-separated fields", lineNumber+1)
		}
		domain := strings.TrimSpace(fields[0])
		includeSubdomains := strings.ToUpper(strings.TrimSpace(fields[1]))
		path := strings.TrimSpace(fields[2])
		secure := strings.ToUpper(strings.TrimSpace(fields[3]))
		expires := strings.TrimSpace(fields[4])
		name := strings.TrimSpace(fields[5])
		cookieValue := strings.TrimSpace(fields[6])

		if domain == "" || (includeSubdomains != "TRUE" && includeSubdomains != "FALSE") || (secure != "TRUE" && secure != "FALSE") {
			return "", fmt.Errorf("invalid Netscape cookie metadata on line %d", lineNumber+1)
		}
		if !strings.HasPrefix(path, "/") {
			return "", fmt.Errorf("invalid Netscape cookie path on line %d", lineNumber+1)
		}
		if _, err := strconv.ParseFloat(expires, 64); err != nil {
			return "", fmt.Errorf("invalid Netscape cookie expiry on line %d", lineNumber+1)
		}
		if name == "" || strings.ContainsAny(name, "\t ;=\r\n\x00") || strings.ContainsAny(cookieValue, "\r\n\x00") {
			return "", fmt.Errorf("invalid Netscape cookie value on line %d", lineNumber+1)
		}

		part := name + "=" + cookieValue
		if position, ok := positions[name]; ok {
			parts[position] = part
		} else {
			positions[name] = len(parts)
			parts = append(parts, part)
		}
		sawRecord = true
	}

	if !sawRecord {
		return "", fmt.Errorf("Netscape cookie file contains no records")
	}
	return strings.Join(parts, "; "), nil
}

func validateCookieHeader(value string) error {
	for _, part := range strings.Split(value, ";") {
		item := strings.TrimSpace(part)
		if item == "" {
			continue
		}
		name, _, ok := strings.Cut(item, "=")
		if !ok || strings.TrimSpace(name) == "" || strings.ContainsAny(name, "\t ;=\r\n\x00") {
			return fmt.Errorf("invalid Cookie header entry")
		}
	}
	return nil
}
