// Ported from: packages/provider-utils/src/validate-download-url.ts
package providerutils

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

// ValidateDownloadUrl validates that a URL is safe to download from, blocking
// private/internal addresses to prevent SSRF attacks.
func ValidateDownloadUrl(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return NewDownloadError(DownloadErrorOptions{
			URL:     rawURL,
			Message: fmt.Sprintf("Invalid URL: %s", rawURL),
		})
	}

	// Only allow http and https protocols
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return NewDownloadError(DownloadErrorOptions{
			URL:     rawURL,
			Message: fmt.Sprintf("URL scheme must be http or https, got %s", parsed.Scheme),
		})
	}

	hostname := parsed.Hostname()

	// Block empty hostname
	if hostname == "" {
		return NewDownloadError(DownloadErrorOptions{
			URL:     rawURL,
			Message: "URL must have a hostname",
		})
	}

	// Block localhost and .local domains
	if hostname == "localhost" ||
		strings.HasSuffix(hostname, ".local") ||
		strings.HasSuffix(hostname, ".localhost") {
		return NewDownloadError(DownloadErrorOptions{
			URL:     rawURL,
			Message: fmt.Sprintf("URL with hostname %s is not allowed", hostname),
		})
	}

	// Check for IPv6 addresses (enclosed in brackets in the original URL)
	if strings.HasPrefix(parsed.Host, "[") {
		ipv6 := hostname // url.Parse already strips brackets
		if isPrivateIPv6(ipv6) {
			return NewDownloadError(DownloadErrorOptions{
				URL:     rawURL,
				Message: fmt.Sprintf("URL with IPv6 address [%s] is not allowed", ipv6),
			})
		}
		return nil
	}

	// Check for IPv4 addresses
	if isIPv4(hostname) {
		if isPrivateIPv4(hostname) {
			return NewDownloadError(DownloadErrorOptions{
				URL:     rawURL,
				Message: fmt.Sprintf("URL with IP address %s is not allowed", hostname),
			})
		}
		return nil
	}

	return nil
}

func isIPv4(hostname string) bool {
	parts := strings.Split(hostname, ".")
	if len(parts) != 4 {
		return false
	}
	for _, part := range parts {
		num, err := strconv.Atoi(part)
		if err != nil {
			return false
		}
		if num < 0 || num > 255 {
			return false
		}
		// Verify no leading zeros
		if strconv.Itoa(num) != part {
			return false
		}
	}
	return true
}

func isPrivateIPv4(ip string) bool {
	parts := strings.Split(ip, ".")
	a, _ := strconv.Atoi(parts[0])
	b, _ := strconv.Atoi(parts[1])

	// 0.0.0.0/8
	if a == 0 {
		return true
	}
	// 10.0.0.0/8
	if a == 10 {
		return true
	}
	// 127.0.0.0/8
	if a == 127 {
		return true
	}
	// 169.254.0.0/16
	if a == 169 && b == 254 {
		return true
	}
	// 172.16.0.0/12
	if a == 172 && b >= 16 && b <= 31 {
		return true
	}
	// 192.168.0.0/16
	if a == 192 && b == 168 {
		return true
	}

	return false
}

func isPrivateIPv6(ip string) bool {
	normalized := strings.ToLower(ip)

	// ::1 (loopback)
	if normalized == "::1" {
		return true
	}
	// :: (unspecified)
	if normalized == "::" {
		return true
	}

	// Check for IPv4-mapped addresses (::ffff:x.x.x.x or ::ffff:HHHH:HHHH)
	if strings.HasPrefix(normalized, "::ffff:") {
		mappedPart := normalized[7:]
		// Dotted-decimal form: ::ffff:127.0.0.1
		if isIPv4(mappedPart) {
			return isPrivateIPv4(mappedPart)
		}
		// Hex form: ::ffff:7f00:1
		hexParts := strings.Split(mappedPart, ":")
		if len(hexParts) == 2 {
			high, err1 := strconv.ParseInt(hexParts[0], 16, 64)
			low, err2 := strconv.ParseInt(hexParts[1], 16, 64)
			if err1 == nil && err2 == nil {
				aVal := (high >> 8) & 0xff
				bVal := high & 0xff
				cVal := (low >> 8) & 0xff
				dVal := low & 0xff
				ipv4 := fmt.Sprintf("%d.%d.%d.%d", aVal, bVal, cVal, dVal)
				return isPrivateIPv4(ipv4)
			}
		}
	}

	// fc00::/7 (unique local addresses - fc00:: and fd00::)
	if strings.HasPrefix(normalized, "fc") || strings.HasPrefix(normalized, "fd") {
		return true
	}

	// fe80::/10 (link-local)
	if strings.HasPrefix(normalized, "fe80") {
		return true
	}

	return false
}

// suppress unused import
var _ = net.IP{}
