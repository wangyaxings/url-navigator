package main

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestConfigRejectsUnsafeOriginsAndSecrets(t *testing.T) {
	good := Config{Listen: "127.0.0.1:8087", Database: "/state/nav.db", OwnerEmailFile: "/secrets/owner", GoogleClientID: "client", GoogleClientSecretFile: "/secrets/google", MasterKeyFile: "/secrets/master", ProviderKeyFile: "/secrets/signing", Clients: []ClientConfig{{"pansou", "/secrets/pan", "https://pan.jomon.dev/auth/sso/callback"}, {"quota", "/secrets/quota", "https://quota.jomon.dev/auth/sso/callback"}}, Cards: []Card{{"PanSou", "https://pan.jomon.dev"}}}
	if err := good.validate(); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*Config){func(c *Config) { c.Listen = "0.0.0.0:8087" }, func(c *Config) { c.Cards = []Card{{"bad", "javascript:alert(1)"}} }, func(c *Config) { c.Clients = []ClientConfig{{"pansou", "secret", "https://evil.test/callback"}} }, func(c *Config) { c.MasterKeyFile = "" }, func(c *Config) { c.Clients[1].SecretFile = c.Clients[0].SecretFile }} {
		c := good
		c.Clients = append([]ClientConfig(nil), good.Clients...)
		mutate(&c)
		if err := c.validate(); err == nil {
			t.Fatal("unsafe config accepted")
		}
	}
}

func TestProtocolLoggerOmitsOAuthStateAndSecrets(t *testing.T) {
	var out bytes.Buffer
	logger := protocolLogger(&out)
	logger.Warn("request rejected", slog.Group("oidc_error", slog.String("state", "sensitive-state")), slog.String("token", "sensitive-token"))
	if strings.Contains(out.String(), "sensitive") || !strings.Contains(out.String(), "request rejected") {
		t.Fatal("unsafe protocol logging", out.String())
	}
}
