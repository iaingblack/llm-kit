package llmkit

import (
	"net"
	"net/url"
	"strings"
)

func IsVercelGateway(rawURL string) bool {
	u, ok := parseBaseURL(rawURL)
	if !ok {
		return false
	}
	return strings.EqualFold(u.Scheme, "https") &&
		strings.EqualFold(u.Hostname(), "ai-gateway.vercel.sh") &&
		strings.TrimRight(u.EscapedPath(), "/") == "/v1"
}

func IsLocalProvider(rawURL string) bool {
	u, ok := parseBaseURL(rawURL)
	if !ok {
		return false
	}
	host := strings.ToLower(u.Hostname())
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func IsAllowedBaseURL(rawURL string) bool {
	u, ok := parseBaseURL(rawURL)
	if !ok {
		return false
	}
	switch strings.ToLower(u.Scheme) {
	case "http":
		return IsLocalProvider(rawURL)
	case "https":
		if IsLocalProvider(rawURL) {
			return true
		}
		ip := net.ParseIP(u.Hostname())
		return ip == nil || isPublicIP(ip)
	default:
		return false
	}
}

func SameURLHost(a, b string) bool {
	ua, okA := parseBaseURL(a)
	ub, okB := parseBaseURL(b)
	if !okA || !okB {
		return false
	}
	return strings.EqualFold(ua.Hostname(), ub.Hostname())
}

func endpointURL(baseURL, path string) string {
	return strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(path, "/")
}

func parseBaseURL(rawURL string) (*url.URL, bool) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil, false
	}
	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme == "" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return nil, false
	}
	if strings.Contains(u.Host, "@") {
		return nil, false
	}
	return u, true
}

func isPublicIP(ip net.IP) bool {
	return ip.IsGlobalUnicast() &&
		!ip.IsPrivate() &&
		!ip.IsLoopback() &&
		!ip.IsLinkLocalUnicast() &&
		!ip.IsLinkLocalMulticast() &&
		!ip.IsUnspecified() &&
		!ip.IsMulticast()
}
