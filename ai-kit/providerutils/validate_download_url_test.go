// Ported from: packages/provider-utils/src/validate-download-url.test.ts
package providerutils

import "testing"

func TestValidateDownloadUrl_AllowedURLs(t *testing.T) {
	tests := []string{
		"https://example.com/image.png",
		"http://example.com/image.png",
		"https://203.0.113.1/file",
		"https://example.com:8080/file",
	}
	for _, url := range tests {
		t.Run(url, func(t *testing.T) {
			if err := ValidateDownloadUrl(url); err != nil {
				t.Errorf("expected no error for %s, got %v", url, err)
			}
		})
	}
}

func TestValidateDownloadUrl_BlockedProtocols(t *testing.T) {
	tests := []string{
		"file:///etc/passwd",
		"ftp://example.com/file",
		"javascript:alert(1)",
		"data:text/plain,hello",
	}
	for _, url := range tests {
		t.Run(url, func(t *testing.T) {
			if err := ValidateDownloadUrl(url); err == nil {
				t.Errorf("expected error for %s", url)
			}
		})
	}
}

func TestValidateDownloadUrl_MalformedURLs(t *testing.T) {
	if err := ValidateDownloadUrl("not-a-url"); err == nil {
		t.Error("expected error for malformed URL")
	}
}

func TestValidateDownloadUrl_BlockedHostnames(t *testing.T) {
	tests := []string{
		"http://localhost/file",
		"http://localhost:3000/file",
		"http://myhost.local/file",
		"http://app.localhost/file",
	}
	for _, url := range tests {
		t.Run(url, func(t *testing.T) {
			if err := ValidateDownloadUrl(url); err == nil {
				t.Errorf("expected error for %s", url)
			}
		})
	}
}

func TestValidateDownloadUrl_BlockedIPv4(t *testing.T) {
	tests := []string{
		"http://127.0.0.1/file",
		"http://127.255.0.1/file",
		"http://10.0.0.1/file",
		"http://172.16.0.1/file",
		"http://172.31.255.255/file",
		"http://192.168.1.1/file",
		"http://169.254.169.254/latest/meta-data/",
		"http://0.0.0.0/file",
	}
	for _, url := range tests {
		t.Run(url, func(t *testing.T) {
			if err := ValidateDownloadUrl(url); err == nil {
				t.Errorf("expected error for %s", url)
			}
		})
	}
}

func TestValidateDownloadUrl_AllowedPublicIPv4(t *testing.T) {
	tests := []string{
		"http://172.15.0.1/file",
		"http://172.32.0.1/file",
	}
	for _, url := range tests {
		t.Run(url, func(t *testing.T) {
			if err := ValidateDownloadUrl(url); err != nil {
				t.Errorf("expected no error for %s, got %v", url, err)
			}
		})
	}
}

func TestValidateDownloadUrl_BlockedIPv6(t *testing.T) {
	tests := []string{
		"http://[::1]/file",
		"http://[::]/file",
		"http://[fc00::1]/file",
		"http://[fd12::1]/file",
		"http://[fe80::1]/file",
	}
	for _, url := range tests {
		t.Run(url, func(t *testing.T) {
			if err := ValidateDownloadUrl(url); err == nil {
				t.Errorf("expected error for %s", url)
			}
		})
	}
}

func TestValidateDownloadUrl_IPv4MappedIPv6_Blocked(t *testing.T) {
	tests := []string{
		"http://[::ffff:127.0.0.1]/file",
		"http://[::ffff:10.0.0.1]/file",
		"http://[::ffff:169.254.169.254]/file",
	}
	for _, url := range tests {
		t.Run(url, func(t *testing.T) {
			if err := ValidateDownloadUrl(url); err == nil {
				t.Errorf("expected error for %s", url)
			}
		})
	}
}

func TestValidateDownloadUrl_IPv4MappedIPv6_Allowed(t *testing.T) {
	if err := ValidateDownloadUrl("http://[::ffff:203.0.113.1]/file"); err != nil {
		t.Errorf("expected no error for public IP, got %v", err)
	}
}
