package netutil

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// IsPrivateOrReservedIP checks if an IP address is in a private, reserved, or link-local range.
func IsPrivateOrReservedIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
		return true
	}
	// Cloud metadata endpoint
	if ip.Equal(net.ParseIP("169.254.169.254")) {
		return true
	}
	// RFC1918
	if ip4 := ip.To4(); ip4 != nil {
		if ip4[0] == 10 {
			return true
		}
		if ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31 {
			return true
		}
		if ip4[0] == 192 && ip4[1] == 168 {
			return true
		}
		return false
	}
	// IPv6 ULA (fc00::/7)
	if len(ip) == net.IPv6len && ip[0]&0xfe == 0xfc {
		return true
	}
	return false
}

// IsURLSafeToProxy checks if a URL targets a safe (non-private) host.
// Resolves the hostname and blocks private/reserved IPs to prevent SSRF attacks.
func IsURLSafeToProxy(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("only http/https URLs are allowed")
	}
	hostname := u.Hostname()
	if hostname == "" {
		return fmt.Errorf("URL must have a hostname")
	}
	// Quick block for obviously private hostnames
	switch strings.ToLower(hostname) {
	case "localhost", "0.0.0.0", "metadata.google.internal":
		return fmt.Errorf("URL targets a blocked host")
	}
	// Resolve hostname and check all resulting IPs
	ips, err := net.LookupIP(hostname)
	if err != nil {
		return fmt.Errorf("could not resolve hostname")
	}
	if len(ips) == 0 {
		return fmt.Errorf("hostname resolved to no addresses")
	}
	for _, ip := range ips {
		if IsPrivateOrReservedIP(ip) {
			return fmt.Errorf("URL targets a private/reserved network address")
		}
	}
	return nil
}
