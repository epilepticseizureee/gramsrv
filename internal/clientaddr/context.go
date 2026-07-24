package clientaddr

import (
	"context"
	"net"
	"net/netip"
	"strings"
)

type remoteAddrKey struct{}

// WithRemoteAddr attaches the physical peer address to the connection context.
func WithRemoteAddr(ctx context.Context, remote string) context.Context {
	return context.WithValue(ctx, remoteAddrKey{}, strings.TrimSpace(remote))
}

// RemoteAddr returns the physical peer address attached by the MTProto edge.
func RemoteAddr(ctx context.Context) (string, bool) {
	remote, ok := ctx.Value(remoteAddrKey{}).(string)
	return remote, ok && remote != ""
}

// RemoteIP extracts and normalizes the IP from the physical peer address.
func RemoteIP(ctx context.Context) (string, bool) {
	remote, ok := RemoteAddr(ctx)
	if !ok {
		return "", false
	}
	return ParseIP(remote)
}

// ParseIP accepts a host:port endpoint or a bare IPv4/IPv6 address.
func ParseIP(remote string) (string, bool) {
	remote = strings.TrimSpace(remote)
	if remote == "" {
		return "", false
	}
	host := remote
	if splitHost, _, err := net.SplitHostPort(remote); err == nil {
		host = splitHost
	}
	host = strings.Trim(host, "[]")
	addr, err := netip.ParseAddr(host)
	if err != nil {
		return "", false
	}
	return addr.Unmap().String(), true
}
