package http

import (
	"net"
	"net/http"
	"net/netip"
	"strings"
)

func clientKey(r *http.Request, trusted []netip.Prefix) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		parts := strings.Split(fwd, ",")
		if len(trusted) == 0 {
			return strings.TrimSpace(parts[0])
		}
		for i := len(parts) - 1; i >= 0; i-- {
			entry := strings.TrimSpace(parts[i])
			if !isTrustedProxy(entry, trusted) {
				return entry
			}
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func isTrustedProxy(entry string, trusted []netip.Prefix) bool {
	addr, err := netip.ParseAddr(entry)
	if err != nil {
		return false
	}
	addr = addr.Unmap()
	if addr.IsPrivate() || addr.IsLoopback() {
		return true
	}
	for _, p := range trusted {
		if p.Contains(addr) {
			return true
		}
	}
	return false
}
