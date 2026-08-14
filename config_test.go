package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	c := defaultConfig()
	if c.Port != 8443 {
		t.Errorf("Port = %d, want 8443", c.Port)
	}
	if !c.HTTPSEnabled {
		t.Error("HTTPSEnabled = false, want true")
	}
	if c.TLSCert != "certs/server.crt" || c.TLSKey != "certs/server.key" {
		t.Errorf("TLS paths = %q/%q", c.TLSCert, c.TLSKey)
	}
	if len(c.Fields) != 7 {
		t.Errorf("Fields len = %d, want 7", len(c.Fields))
	}
	if len(c.AllowedNetworks.IPv4) == 0 {
		t.Error("AllowedNetworks.IPv4 empty")
	}
}

func TestLoadConfigMissing(t *testing.T) {
	c, err := LoadConfig(filepath.Join(t.TempDir(), "nope.yaml"))
	if err == nil {
		t.Error("expected error for missing config, got nil")
	}
	if c == nil {
		t.Fatal("expected default config (non-nil)")
	}
	if c.Port != 8443 {
		t.Errorf("fallback Port = %d, want 8443", c.Port)
	}
}

func TestLoadConfigValid(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.yaml")
	content := `app_name: "Test"
port: 9000
https_enabled: false
tls_cert: "certs/x.crt"
tls_key: "certs/x.key"
fields:
  - key: name
    label: 姓名
    maxlen: 20
`
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	c, err := LoadConfig(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Port != 9000 || c.AppName != "Test" || c.HTTPSEnabled {
		t.Errorf("parsed config wrong: %+v", c)
	}
	if len(c.Fields) != 1 || c.Fields[0].Key != "name" {
		t.Errorf("fields parse wrong: %+v", c.Fields)
	}
}

func TestLoadConfigInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(p, []byte("port: : : bad\n"), 0644); err != nil {
		t.Fatal(err)
	}
	c, err := LoadConfig(p)
	if err == nil {
		t.Error("expected error for invalid yaml")
	}
	if c == nil || c.Port != 8443 {
		t.Errorf("expected default fallback config, got %+v", c)
	}
}
