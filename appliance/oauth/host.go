package oauth

import (
	"net"
	"net/http"
	"strconv"
	"strings"
)

// GuardHost rejects Host headers that cannot be safely embedded in URLs,
// headers, or JSON. It is middleware-framework agnostic: wrap any http.Handler.
func GuardHost(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !ValidRequestHost(r.Host) {
			http.Error(w, "invalid Host header", http.StatusBadRequest)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ValidRequestHost accepts a DNS name, IPv4 literal, or bracketed IPv6 literal
// with an optional digits-only port. Rejected characters cannot break out of
// a URL, header value, or JSON string.
func ValidRequestHost(host string) bool {
	if host == "" || len(host) > 253 {
		return false
	}
	if strings.HasPrefix(host, "[") {
		end := strings.Index(host, "]")
		if end < 0 || net.ParseIP(host[1:end]) == nil {
			return false
		}
		rest := host[end+1:]
		if rest == "" {
			return true
		}
		return strings.HasPrefix(rest, ":") && isPort(rest[1:])
	}
	if strings.Contains(host, ":") {
		name, port, err := net.SplitHostPort(host)
		if err != nil || !isPort(port) {
			return false
		}
		host = name
	}
	if host == "" || strings.HasPrefix(host, ".") || strings.HasSuffix(host, ".") || strings.Contains(host, "..") {
		return false
	}
	for _, c := range host {
		valid := c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' ||
			c >= '0' && c <= '9' || c == '-' || c == '.' || c == '_'
		if !valid {
			return false
		}
	}
	return true
}

// ValidPort reports whether s is a decimal TCP port in 1..65535.
func ValidPort(s string) bool { return isPort(s) }

func isPort(s string) bool {
	if len(s) == 0 || len(s) > 5 {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	port, err := strconv.Atoi(s)
	return err == nil && port >= 1 && port <= 65535
}
