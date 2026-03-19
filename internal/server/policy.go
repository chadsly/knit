package server

import (
	"net"
	"net/url"
	"strings"

	"knit/internal/config"
	"knit/internal/redaction"
	"knit/internal/session"
)

func endpointAllowedByPolicy(endpoint string, cfg config.Config) bool {
	if strings.TrimSpace(endpoint) == "" {
		return true
	}
	if !isSecureEndpoint(endpoint) {
		return false
	}
	return redaction.URLAllowed(endpoint, cfg.OutboundAllowlist, cfg.BlockedTargets)
}

func isSecureEndpoint(endpoint string) bool {
	u, err := url.Parse(strings.TrimSpace(endpoint))
	if err != nil {
		return false
	}
	switch strings.ToLower(u.Scheme) {
	case "https":
		return true
	case "http":
		host := u.Hostname()
		if host == "" {
			return false
		}
		if strings.EqualFold(host, "localhost") {
			return true
		}
		if ip := net.ParseIP(host); ip != nil {
			return ip.IsLoopback()
		}
		return false
	default:
		return false
	}
}

func redactPackageForTransmission(pkg session.CanonicalPackage) session.CanonicalPackage {
	out := pkg
	out.Summary = redaction.Text(out.Summary)
	out.ChangeRequests = append([]session.ChangeReq(nil), pkg.ChangeRequests...)
	for i := range out.ChangeRequests {
		out.ChangeRequests[i].Summary = redaction.Text(out.ChangeRequests[i].Summary)
	}
	return out
}
