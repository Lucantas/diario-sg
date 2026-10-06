package http

import (
	"net"
	"net/http"
	"strings"
)

func clientKey(r *http.Request, hops int) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		parts := strings.Split(fwd, ",")
		switch {
		case hops == 0:
			return strings.TrimSpace(parts[0])
		case hops <= len(parts):
			return strings.TrimSpace(parts[len(parts)-hops])
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
