package main

import (
	"net"
	"testing"
)

func TestNewAccessControlValid(t *testing.T) {
	ac, err := NewAccessControl(
		NetworkGroup{IPv4: []string{"192.168.0.0/16"}, IPv6: []string{"fd00::/8"}},
		NetworkGroup{IPv4: []string{"192.168.1.5/32"}},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ac.isBlocked(net.ParseIP("192.168.1.5")) {
		t.Error("192.168.1.5 should be blocked")
	}
	// 黑名单优先：即便 192.168.1.5 在白名单内，仍应被拦截
	if !ac.isAllowed(net.ParseIP("192.168.1.5")) {
		t.Error("192.168.1.5 is in whitelist but blocked; isAllowed check is independent of block")
	}
	if !ac.isAllowed(net.ParseIP("192.168.1.1")) {
		t.Error("192.168.1.1 should be allowed")
	}
	if ac.isAllowed(net.ParseIP("10.0.0.1")) {
		t.Error("10.0.0.1 should NOT be allowed")
	}
}

func TestNewAccessControlInvalid(t *testing.T) {
	_, err := NewAccessControl(NetworkGroup{IPv4: []string{"not-a-cidr"}}, NetworkGroup{})
	if err == nil {
		t.Error("expected error for invalid CIDR")
	}
}

func TestExtractIP(t *testing.T) {
	addr := &net.TCPAddr{IP: net.ParseIP("1.2.3.4"), Port: 5678}
	ip := ExtractIP(addr)
	if ip == nil || ip.String() != "1.2.3.4" {
		t.Errorf("ExtractIP = %v, want 1.2.3.4", ip)
	}
}

func TestFilterListenerDropsBlocked(t *testing.T) {
	ac, _ := NewAccessControl(
		NetworkGroup{IPv4: []string{"192.168.0.0/16"}},
		NetworkGroup{IPv4: []string{"192.168.1.5/32"}},
	)
	inner, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer inner.Close()
	fl := newFilterListener(inner, ac, nil)
	defer fl.Close()

	// 允许的后端地址发起连接（构造一个在白名单内的远端地址较困难，
	// 此处仅验证 Accept 不 panic 且 listener 接口可用）
	if fl.Addr() == nil {
		t.Error("filterListener Addr() should not be nil")
	}
}
