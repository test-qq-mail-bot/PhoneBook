package main

import (
	"fmt"
	"net"
	"sync"
)

// AccessControl 访问控制（CIDR 黑白名单，黑名单优先）
type AccessControl struct {
	mu       sync.RWMutex
	allowed  []*net.IPNet
	blocked  []*net.IPNet
	allowRaw []string
	blockRaw []string
}

// NewAccessControl 解析网段配置
func NewAccessControl(allowed, blocked NetworkGroup) (*AccessControl, error) {
	ac := &AccessControl{allowRaw: allowed.IPv4, blockRaw: blocked.IPv4}
	if err := ac.addGroup(allowed.IPv4, &ac.allowed); err != nil {
		return nil, err
	}
	if err := ac.addGroup(allowed.IPv6, &ac.allowed); err != nil {
		return nil, err
	}
	if err := ac.addGroup(blocked.IPv4, &ac.blocked); err != nil {
		return nil, err
	}
	if err := ac.addGroup(blocked.IPv6, &ac.blocked); err != nil {
		return nil, err
	}
	return ac, nil
}

func (ac *AccessControl) addGroup(cidrs []string, dst *[]*net.IPNet) error {
	for _, c := range cidrs {
		if c == "" {
			continue
		}
		// 兼容裸 IP 地址（如 "127.0.0.1"），自动归一化为 CIDR 格式
		if _, ipnet, err := net.ParseCIDR(c); err == nil {
			*dst = append(*dst, ipnet)
			continue
		}
		if ip := net.ParseIP(c); ip != nil {
			if ip4 := ip.To4(); ip4 != nil {
				_, ipnet, _ := net.ParseCIDR(ip4.String() + "/32")
				*dst = append(*dst, ipnet)
			} else {
				_, ipnet, _ := net.ParseCIDR(ip.String() + "/128")
				*dst = append(*dst, ipnet)
			}
			continue
		}
		return fmt.Errorf("无效的网段或 IP %q", c)
	}
	return nil
}

func (ac *AccessControl) isBlocked(ip net.IP) bool {
	ac.mu.RLock()
	defer ac.mu.RUnlock()
	for _, n := range ac.blocked {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

func (ac *AccessControl) isAllowed(ip net.IP) bool {
	ac.mu.RLock()
	defer ac.mu.RUnlock()
	for _, n := range ac.allowed {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// ExtractIP 从 RemoteAddr 提取 IP（去掉端口）
func ExtractIP(addr net.Addr) net.IP {
	host, _, err := net.SplitHostPort(addr.String())
	if err != nil {
		return net.ParseIP(addr.String())
	}
	return net.ParseIP(host)
}

// filterListener 在 TCP 层对连接做黑白名单过滤：
// 命中黑名单 或 不在白名单 → 立即关闭连接（不发送任何应用层字节）。
type filterListener struct {
	inner  net.Listener
	ac     *AccessControl
	onDrop func(ip net.IP, reason string)
}

func newFilterListener(inner net.Listener, ac *AccessControl, onDrop func(net.IP, string)) *filterListener {
	return &filterListener{inner: inner, ac: ac, onDrop: onDrop}
}

func (f *filterListener) Accept() (net.Conn, error) {
	for {
		conn, err := f.inner.Accept()
		if err != nil {
			return nil, err
		}
		ip := ExtractIP(conn.RemoteAddr())
		if ip == nil {
			conn.Close()
			continue
		}
		if f.ac.isBlocked(ip) {
			if f.onDrop != nil {
				f.onDrop(ip, "黑名单")
			}
			conn.Close()
			continue
		}
		if !f.ac.isAllowed(ip) {
			if f.onDrop != nil {
				f.onDrop(ip, "不在允许网段内")
			}
			conn.Close()
			continue
		}
		return conn, nil
	}
}

func (f *filterListener) Close() error   { return f.inner.Close() }
func (f *filterListener) Addr() net.Addr { return f.inner.Addr() }
