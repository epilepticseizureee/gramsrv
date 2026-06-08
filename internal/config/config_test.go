package config

import (
	"net"
	"testing"
)

func TestLoadUsesExplicitAdvertiseIP(t *testing.T) {
	t.Setenv("TELESRV_ADVERTISE_IP", "10.172.61.102")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.AdvertiseIP != "10.172.61.102" {
		t.Fatalf("AdvertiseIP = %q, want explicit env", cfg.AdvertiseIP)
	}
}

func TestPreferredAdvertiseIPPrefersPhysicalLANRanges(t *testing.T) {
	got := preferredAdvertiseIP([]net.IP{
		net.IPv4(172, 17, 0, 1),
		net.IPv4(192, 168, 1, 20),
		net.IPv4(10, 172, 61, 102),
	})
	if got != "10.172.61.102" {
		t.Fatalf("preferredAdvertiseIP = %q, want 10.172.61.102", got)
	}

	got = preferredAdvertiseIP([]net.IP{
		net.IPv4(172, 17, 0, 1),
		net.IPv4(192, 168, 1, 20),
	})
	if got != "192.168.1.20" {
		t.Fatalf("preferredAdvertiseIP = %q, want 192.168.1.20", got)
	}
}

func TestPrivateIPv4AndVirtualInterfaceDetection(t *testing.T) {
	if !isPrivateIPv4(net.IPv4(10, 172, 61, 102)) {
		t.Fatal("10.172.61.102 should be private")
	}
	if isPrivateIPv4(net.IPv4(8, 8, 8, 8)) {
		t.Fatal("8.8.8.8 should not be private")
	}
	if !likelyVirtualInterface("vEthernet (WSL)") {
		t.Fatal("vEthernet (WSL) should be treated as virtual")
	}
}
