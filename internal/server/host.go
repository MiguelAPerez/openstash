package server

import (
	"fmt"
	"net"
	"net/http"
	"strings"
)

// hostPolicy decides which Host headers the server answers. It blocks DNS
// rebinding: a page on an attacker's domain can re-point that name at
// 127.0.0.1 and then call a local server as same-origin (so CORS and the
// JSON-only POST check don't apply), but the browser still sends the
// attacker's name in Host.
//
// Always allowed: localhost, *.localhost and IP literals (an IP can't be
// rebound). Also allowed: the host the server is bound to and any configured
// names. A wildcard bind (":8080", "0.0.0.0", "::") with nothing configured
// accepts any Host, so Docker and LAN setups reached by hostname keep working;
// "*" in the configured list does the same for any bind.
type hostPolicy struct {
	any   bool
	names map[string]bool
}

func newHostPolicy(addr string, allowed []string) hostPolicy {
	p := hostPolicy{names: map[string]bool{"localhost": true}}
	configured := false
	for _, h := range allowed {
		h = normalizeHost(h)
		if h == "" {
			continue
		}
		if h == "*" {
			return hostPolicy{any: true}
		}
		p.names[h] = true
		configured = true
	}
	bind, _, err := net.SplitHostPort(addr)
	if err != nil {
		bind = addr
	}
	bind = normalizeHost(bind)
	if bind == "" || bind == "0.0.0.0" || bind == "::" {
		if !configured {
			return hostPolicy{any: true}
		}
	} else {
		p.names[bind] = true
	}
	return p
}

func (p hostPolicy) allows(hostHeader string) bool {
	if p.any {
		return true
	}
	h := hostHeader
	if host, _, err := net.SplitHostPort(hostHeader); err == nil {
		h = host
	}
	h = normalizeHost(h)
	if p.names[h] || strings.HasSuffix(h, ".localhost") {
		return true
	}
	return net.ParseIP(h) != nil
}

func (p hostPolicy) wrap(next http.Handler) http.Handler {
	if p.any {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !p.allows(r.Host) {
			writeError(w, http.StatusForbidden, fmt.Sprintf(
				"host %q is not allowed; add it with --allowed-hosts or OPENSTASH_ALLOWED_HOSTS", r.Host))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func normalizeHost(h string) string {
	h = strings.ToLower(strings.TrimSpace(h))
	h = strings.TrimSuffix(strings.TrimPrefix(h, "["), "]")
	return strings.TrimSuffix(h, ".")
}
