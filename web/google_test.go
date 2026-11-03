package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	googleoidc "github.com/coreos/go-oidc/v3/oidc"
	jose "github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"golang.org/x/oauth2"
)

func TestGoogleRejectsWrongSignatureIssuerAudienceExpiryNonce(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	other, _ := rsa.GenerateKey(rand.Reader, 2048)
	var raw string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/keys" {
			json.NewEncoder(w).Encode(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{Key: &key.PublicKey, KeyID: "google", Algorithm: "RS256", Use: "sig"}}})
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"access_token": "unused", "token_type": "Bearer", "id_token": raw})
	}))
	defer srv.Close()
	g := &googleLogin{config: oauth2.Config{ClientID: "google-client", ClientSecret: "secret", Endpoint: oauth2.Endpoint{TokenURL: srv.URL + "/token", AuthStyle: oauth2.AuthStyleInParams}}, verifier: googleoidc.NewVerifier("https://accounts.google.com", googleoidc.NewRemoteKeySet(context.Background(), srv.URL+"/keys"), &googleoidc.Config{ClientID: "google-client", SupportedSigningAlgs: []string{"RS256"}})}
	for _, test := range []string{"valid", "signature", "issuer", "audience", "expiry", "nonce"} {
		t.Run(test, func(t *testing.T) {
			claims := map[string]any{"iss": "https://accounts.google.com", "sub": "owner-sub", "aud": "google-client", "exp": time.Now().Add(time.Hour).Unix(), "iat": time.Now().Unix(), "nonce": "expected", "email": "owner@example.test", "email_verified": true}
			signing := key
			switch test {
			case "signature":
				signing = other
			case "issuer":
				claims["iss"] = "https://evil.test"
			case "audience":
				claims["aud"] = "another-client"
			case "expiry":
				claims["exp"] = time.Now().Add(-time.Hour).Unix()
			case "nonce":
				claims["nonce"] = "wrong"
			}
			signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: signing}, (&jose.SignerOptions{}).WithHeader("kid", "google"))
			if err != nil {
				t.Fatal(err)
			}
			raw, err = jwt.Signed(signer).Claims(claims).Serialize()
			if err != nil {
				t.Fatal(err)
			}
			identity, err := g.exchange(context.Background(), "code", &Handshake{Nonce: "expected", Verifier: oauth2.GenerateVerifier()})
			if test == "valid" {
				if err != nil || identity.Subject != "owner-sub" || !identity.Verified {
					t.Fatal("valid identity rejected", err)
				}
			} else if err == nil {
				t.Fatal("invalid upstream JWT accepted")
			}
		})
	}
}
