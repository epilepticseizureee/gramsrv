package clientaddr

import (
	"context"
	"testing"
)

func TestRemoteIP(t *testing.T) {
	tests := []struct {
		name   string
		remote string
		want   string
		ok     bool
	}{
		{name: "ipv4 endpoint", remote: "203.0.113.7:2398", want: "203.0.113.7", ok: true},
		{name: "ipv6 endpoint", remote: "[2001:db8::7]:2398", want: "2001:db8::7", ok: true},
		{name: "bare ipv4", remote: "198.51.100.9", want: "198.51.100.9", ok: true},
		{name: "invalid", remote: "not-an-address", ok: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := RemoteIP(WithRemoteAddr(context.Background(), tt.remote))
			if got != tt.want || ok != tt.ok {
				t.Fatalf("RemoteIP(%q) = %q, %v; want %q, %v", tt.remote, got, ok, tt.want, tt.ok)
			}
		})
	}
}
