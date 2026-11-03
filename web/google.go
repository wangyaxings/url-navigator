package main

import (
	"context"
	"net/http"
	"time"

	googleoidc "github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

type upstream interface {
	loginURL(*Handshake) string
	exchange(context.Context, string, *Handshake) (Identity, error)
}
type googleLogin struct {
	config   oauth2.Config
	verifier *googleoidc.IDTokenVerifier
}

func newGoogle(ctx context.Context, id, secret string) (*googleLogin, error) {
	ctx = googleoidc.ClientContext(ctx, &http.Client{Timeout: 15 * time.Second})
	p, err := googleoidc.NewProvider(ctx, "https://accounts.google.com")
	if err != nil {
		return nil, err
	}
	return &googleLogin{config: oauth2.Config{ClientID: id, ClientSecret: secret, Endpoint: p.Endpoint(), RedirectURL: issuer + "/auth/google/callback", Scopes: []string{"openid", "email", "profile"}}, verifier: p.Verifier(&googleoidc.Config{ClientID: id, SupportedSigningAlgs: []string{"RS256"}})}, nil
}
func (g *googleLogin) loginURL(h *Handshake) string {
	return g.config.AuthCodeURL(h.State, oauth2.SetAuthURLParam("nonce", h.Nonce), oauth2.S256ChallengeOption(h.Verifier), oauth2.SetAuthURLParam("prompt", "select_account"))
}
func (g *googleLogin) exchange(ctx context.Context, code string, h *Handshake) (Identity, error) {
	ctx = context.WithValue(ctx, oauth2.HTTPClient, &http.Client{Timeout: 15 * time.Second})
	token, err := g.config.Exchange(ctx, code, oauth2.VerifierOption(h.Verifier))
	if err != nil {
		return Identity{}, errDenied
	}
	raw, ok := token.Extra("id_token").(string)
	if !ok {
		return Identity{}, errDenied
	}
	verified, err := g.verifier.Verify(ctx, raw)
	if err != nil || !constantEqual(verified.Nonce, h.Nonce) {
		return Identity{}, errDenied
	}
	var claims struct {
		Email    string `json:"email"`
		Verified bool   `json:"email_verified"`
	}
	if err = verified.Claims(&claims); err != nil {
		return Identity{}, errDenied
	}
	return Identity{Subject: verified.Subject, Email: claims.Email, Verified: claims.Verified}, nil
}
