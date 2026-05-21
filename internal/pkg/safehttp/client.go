// Package safehttp provides an HTTP client with SSRF protection.
// It validates resolved IP addresses before connecting, blocking
// requests to loopback, link-local, private RFC1918, IPv6 ULA,
// and cloud metadata endpoints.
package safehttp

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

// blockedIP returns an error if the IP address should not be reachable
// from an outbound HTTP request. This prevents SSRF attacks where an
// attacker tricks the server into making requests to internal services.
func blockedIP(ip net.IP) error {
	if ip.IsLoopback() {
		return fmt.Errorf("blocked: loopback address %s", ip)
	}

	// Block link-local (169.254.0.0/16, fe80::/10) — includes cloud metadata endpoints
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return fmt.Errorf("blocked: link-local address %s", ip)
	}

	// Block private RFC1918 (10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16)
	// and IPv6 ULA (fd00::/8)
	if ip.IsPrivate() {
		return fmt.Errorf("blocked: private address %s", ip)
	}

	// Block unspecified (0.0.0.0, ::)
	if ip.IsUnspecified() {
		return fmt.Errorf("blocked: unspecified address %s", ip)
	}

	return nil
}

// SafeTransport wraps http.Transport and validates resolved IP addresses
// before establishing connections. This prevents SSRF attacks by blocking
// requests to internal/private network ranges.
type SafeTransport struct {
	*http.Transport
}

// NewSafeTransport creates a new SafeTransport with default settings.
func NewSafeTransport() *SafeTransport {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DialContext = safeDialContext(transport)
	return &SafeTransport{Transport: transport}
}

// safeDialContext returns a DialContext function that validates resolved
// IPs before connecting. It uses the underlying transport's resolver.
func safeDialContext(transport *http.Transport) func(ctx context.Context, network, addr string) (net.Conn, error) {
	// Capture the original dialer
	originalDialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, fmt.Errorf("invalid address %q: %w", addr, err)
		}

		// Resolve the hostname to IP addresses
		resolver := &net.Resolver{}
		ips, err := resolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve %q: %w", host, err)
		}

		if len(ips) == 0 {
			return nil, fmt.Errorf("no addresses found for %q", host)
		}

		// Validate each resolved IP
		for _, ipAddr := range ips {
			ip := ipAddr.IP
			if err := blockedIP(ip); err != nil {
				return nil, fmt.Errorf("SSRF protection: %w", err)
			}
		}

		// All IPs passed validation — connect using the first one
		// (or let the dialer handle Happy Eyeballs if multiple)
		return originalDialer.DialContext(ctx, network, net.JoinHostPort(ips[0].String(), port))
	}
}

// NewSafeClient creates an HTTP client with SSRF protection.
// The client uses a SafeTransport that validates resolved IP addresses
// before connecting, preventing requests to internal/private networks.
func NewSafeClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Transport: NewSafeTransport(),
		Timeout:   timeout,
	}
}
