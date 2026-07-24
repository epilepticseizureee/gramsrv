package rpc

import (
	"context"

	"telesrv/internal/geoip"
)

func (r *Router) geoIPLocation(ctx context.Context, ip string) (geoip.Location, bool) {
	if r.deps.GeoIP == nil {
		return geoip.Location{}, false
	}
	lang := ""
	if info, ok := ClientInfoFrom(ctx); ok {
		lang = info.LangCode
	}
	return r.deps.GeoIP.Lookup(ip, lang)
}
