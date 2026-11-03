package main

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

type ClientConfig struct {
	ID         string `json:"id"`
	SecretFile string `json:"secret_file"`
	Redirect   string `json:"redirect_uri"`
}
type Config struct {
	Listen                 string         `json:"listen"`
	Database               string         `json:"database"`
	OwnerEmailFile         string         `json:"owner_email_file"`
	GoogleClientID         string         `json:"google_client_id"`
	GoogleClientSecretFile string         `json:"google_client_secret_file"`
	MasterKeyFile          string         `json:"master_key_file"`
	ProviderKeyFile        string         `json:"provider_key_file"`
	Clients                []ClientConfig `json:"clients"`
	Cards                  []Card         `json:"cards"`
}

func (c *Config) validate() error {
	host, port, err := net.SplitHostPort(c.Listen)
	ip := net.ParseIP(host)
	if err != nil || ip == nil || !ip.IsLoopback() || port == "" {
		return errors.New("listen must be a loopback IP and port")
	}
	if c.Database == "" || c.OwnerEmailFile == "" || c.GoogleClientID == "" || c.GoogleClientSecretFile == "" || c.MasterKeyFile == "" || c.ProviderKeyFile == "" {
		return errors.New("all state and secret paths are required")
	}
	if len(c.Clients) != 2 {
		return errors.New("exactly pansou and quota clients are required")
	}
	seen := map[string]bool{}
	files := map[string]bool{}
	for _, client := range c.Clients {
		want := map[string]string{"pansou": "https://pan.jomon.dev/auth/sso/callback", "quota": "https://quota.jomon.dev/auth/sso/callback"}[client.ID]
		if want == "" || client.Redirect != want || seen[client.ID] || client.SecretFile == "" || files[client.SecretFile] {
			return errors.New("invalid client registration")
		}
		seen[client.ID] = true
		files[client.SecretFile] = true
	}
	for _, card := range c.Cards {
		u, e := url.Parse(card.URL)
		if e != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || card.Name == "" {
			return errors.New("cards require a name and HTTPS URL")
		}
	}
	return nil
}
func readSecret(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	v := strings.TrimSpace(string(b))
	if v == "" {
		return "", errors.New("empty secret file")
	}
	return v, nil
}
func loadConfig(path string) (Config, error) {
	var c Config
	f, err := os.Open(path)
	if err != nil {
		return c, err
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	dec.DisallowUnknownFields()
	if err = dec.Decode(&c); err != nil {
		return c, err
	}
	if c.Listen == "" {
		c.Listen = "127.0.0.1:8087"
	}
	if len(c.Cards) == 0 {
		c.Cards = []Card{{"PanSou", "https://pan.jomon.dev"}, {"Quota", "https://quota.jomon.dev"}}
	}
	return c, c.validate()
}
func loadSigningKey(path string) (*rsa.PrivateKey, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(b)
	if block == nil {
		return nil, errors.New("invalid signing PEM")
	}
	var key *rsa.PrivateKey
	if block.Type == "RSA PRIVATE KEY" {
		key, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	} else {
		var v any
		v, err = x509.ParsePKCS8PrivateKey(block.Bytes)
		if err == nil {
			var ok bool
			key, ok = v.(*rsa.PrivateKey)
			if !ok {
				return nil, errors.New("RSA signing key required")
			}
		}
	}
	if err != nil {
		return nil, err
	}
	if key.N.BitLen() < 2048 {
		return nil, errors.New("RSA signing key must have at least 2048 bits")
	}
	if err = key.Validate(); err != nil {
		return nil, err
	}
	return key, nil
}
func run() error {
	slog.SetDefault(protocolLogger(os.Stderr))
	configPath := flag.String("config", "/etc/nav/config.json", "path to private configuration")
	flag.Parse()
	c, err := loadConfig(*configPath)
	if err != nil {
		return err
	}
	owner, err := readSecret(c.OwnerEmailFile)
	if err != nil {
		return err
	}
	if !strings.Contains(owner, "@") {
		return errors.New("invalid owner email")
	}
	googleSecret, err := readSecret(c.GoogleClientSecretFile)
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(c.MasterKeyFile)
	if err != nil {
		return err
	}
	if len(raw) != 32 {
		return errors.New("master key file must contain exactly 32 random raw bytes")
	}
	var master [32]byte
	copy(master[:], raw)
	key, err := loadSigningKey(c.ProviderKeyFile)
	if err != nil {
		return err
	}
	s, err := openStore(c.Database, master, owner)
	if err != nil {
		return err
	}
	defer s.db.Close()
	s.key = key
	secrets := map[string]bool{}
	for _, cc := range c.Clients {
		secret, err := readSecret(cc.SecretFile)
		if err != nil {
			return err
		}
		if len(secret) < 32 || secrets[secret] {
			return errors.New("client secrets must be distinct and at least 32 characters")
		}
		secrets[secret] = true
		s.clients[cc.ID] = &Client{ID: cc.ID, Secret: secret, Redirect: cc.Redirect}
	}
	// Detect a wrong master key at startup rather than failing during the owner's login.
	if err = s.update(func(d *diskState) error {
		if d.Owner.Seed != "" {
			_, e := s.unseal(d.Owner.Seed)
			return e
		}
		return nil
	}); err != nil {
		return fmt.Errorf("state decryption failed: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	g, err := newGoogle(ctx, c.GoogleClientID, googleSecret)
	cancel()
	if err != nil {
		return errors.New("Google discovery unavailable")
	}
	_, handler, err := newApp(s, master, g, c.Cards)
	if err != nil {
		return err
	}
	server := &http.Server{Addr: c.Listen, Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 20 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 32 << 10}
	stopped := make(chan os.Signal, 1)
	signal.Notify(stopped, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(stopped)
	go func() {
		<-stopped
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}()
	log.Printf("portal listening on %s", c.Listen)
	if err = server.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Provider diagnostics are intentionally limited to event messages. Its error
// attributes can contain OAuth state or request values and must not reach logs.
func protocolLogger(w io.Writer) *slog.Logger {
	return slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
		if len(groups) == 0 && (a.Key == slog.TimeKey || a.Key == slog.LevelKey || a.Key == slog.MessageKey) {
			return a
		}
		return slog.Attr{}
	}}))
}
func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
