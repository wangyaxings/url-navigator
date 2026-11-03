package main

import (
	"context"
	"crypto/rsa"
	"crypto/subtle"
	"errors"
	"net/url"
	"slices"
	"time"

	jose "github.com/go-jose/go-jose/v4"
	"github.com/zitadel/oidc/v3/pkg/oidc"
	"github.com/zitadel/oidc/v3/pkg/op"
)

type AuthRequest struct {
	ID                         string
	Request                    oidc.AuthRequest
	SessionID, Subject, Code   string
	AuthTime, Created, Expires time.Time
	Consumed                   bool
}

func (a *AuthRequest) GetID() string          { return a.ID }
func (a *AuthRequest) GetACR() string         { return acr }
func (a *AuthRequest) GetAMR() []string       { return []string{"federated", "mfa"} }
func (a *AuthRequest) GetAudience() []string  { return []string{a.Request.ClientID} }
func (a *AuthRequest) GetAuthTime() time.Time { return a.AuthTime }
func (a *AuthRequest) GetClientID() string    { return a.Request.ClientID }
func (a *AuthRequest) GetCodeChallenge() *oidc.CodeChallenge {
	return &oidc.CodeChallenge{Challenge: a.Request.CodeChallenge, Method: a.Request.CodeChallengeMethod}
}
func (a *AuthRequest) GetNonce() string                   { return a.Request.Nonce }
func (a *AuthRequest) GetRedirectURI() string             { return a.Request.RedirectURI }
func (a *AuthRequest) GetResponseType() oidc.ResponseType { return a.Request.ResponseType }
func (a *AuthRequest) GetResponseMode() oidc.ResponseMode { return a.Request.ResponseMode }
func (a *AuthRequest) GetScopes() []string                { return a.Request.Scopes }
func (a *AuthRequest) GetState() string                   { return a.Request.State }
func (a *AuthRequest) GetSubject() string                 { return a.Subject }
func (a *AuthRequest) Done() bool                         { return a.SessionID != "" && a.Subject != "" }

type Client struct{ ID, Secret, Redirect string }

func (c *Client) GetID() string                       { return c.ID }
func (c *Client) RedirectURIs() []string              { return []string{c.Redirect} }
func (c *Client) PostLogoutRedirectURIs() []string    { return nil }
func (c *Client) ApplicationType() op.ApplicationType { return op.ApplicationTypeWeb }
func (c *Client) AuthMethod() oidc.AuthMethod         { return oidc.AuthMethodBasic }
func (c *Client) ResponseTypes() []oidc.ResponseType {
	return []oidc.ResponseType{oidc.ResponseTypeCode}
}
func (c *Client) GrantTypes() []oidc.GrantType        { return []oidc.GrantType{oidc.GrantTypeCode} }
func (c *Client) LoginURL(id string) string           { return "/login?request=" + url.QueryEscape(id) }
func (c *Client) AccessTokenType() op.AccessTokenType { return op.AccessTokenTypeBearer }
func (c *Client) IDTokenLifetime() time.Duration      { return 5 * time.Minute }
func (c *Client) DevMode() bool                       { return false }
func (c *Client) RestrictAdditionalIdTokenScopes() func([]string) []string {
	return func(s []string) []string { return s }
}
func (c *Client) RestrictAdditionalAccessTokenScopes() func([]string) []string {
	return func(s []string) []string { return s }
}
func (c *Client) IsScopeAllowed(s string) bool {
	return s == "openid" || s == "email" || s == "profile"
}
func (c *Client) IDTokenUserinfoClaimsAssertion() bool { return true }
func (c *Client) ClockSkew() time.Duration             { return 0 }
func constantEqual(a, b string) bool                   { return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1 }

func (s *Store) CreateAuthRequest(_ context.Context, r *oidc.AuthRequest, _ string) (op.AuthRequest, error) {
	if r.ResponseType != oidc.ResponseTypeCode || r.CodeChallengeMethod != oidc.CodeChallengeMethodS256 || len(r.CodeChallenge) != 43 || r.State == "" || r.Nonce == "" || !slices.Contains(r.Scopes, "openid") {
		return nil, oidc.ErrInvalidRequest().WithDescription("code, S256, state, nonce and openid are required")
	}
	for _, scope := range r.Scopes {
		if scope != "openid" && scope != "email" && scope != "profile" {
			return nil, oidc.ErrInvalidScope()
		}
	}
	a := &AuthRequest{ID: randomToken(), Request: *r, Created: time.Now(), Expires: time.Now().Add(10 * time.Minute)}
	err := s.update(func(d *diskState) error {
		if len(d.Auth) >= 500 {
			return errDenied
		}
		d.Auth[a.ID] = a
		return nil
	})
	return a, err
}
func (s *Store) AuthRequestByID(_ context.Context, id string) (op.AuthRequest, error) {
	var out AuthRequest
	err := s.update(func(d *diskState) error {
		a := d.Auth[id]
		if a == nil || a.Consumed {
			return errDenied
		}
		if a.Done() && !validSession(d.Sessions[a.SessionID], time.Now()) {
			return errDenied
		}
		out = *a
		return nil
	})
	return &out, err
}
func (s *Store) completeAuth(id, sid string) error {
	return s.update(func(d *diskState) error {
		a := d.Auth[id]
		sess := d.Sessions[sid]
		if a == nil || a.Code != "" || a.Consumed || !validSession(sess, time.Now()) || time.Until(sess.Expires) < 6*time.Minute {
			return errDenied
		}
		if a.Request.MaxAge != nil {
			if *a.Request.MaxAge == 0 {
				if sess.AuthTime.Before(a.Created) {
					return errDenied
				}
			} else if time.Since(sess.AuthTime) > time.Duration(*a.Request.MaxAge)*time.Second {
				return errDenied
			}
		}
		a.SessionID = sid
		a.Subject = d.Owner.ID
		a.AuthTime = sess.AuthTime
		return nil
	})
}
func (s *Store) SaveAuthCode(_ context.Context, id, code string) error {
	return s.update(func(d *diskState) error {
		a := d.Auth[id]
		if a == nil || !a.Done() || a.Code != "" || a.Consumed || !validSession(d.Sessions[a.SessionID], time.Now()) {
			return errDenied
		}
		a.Code = hash(code)
		a.Expires = time.Now().Add(time.Minute)
		return nil
	})
}

// Consume before returning, so simultaneous token requests cannot both succeed.
func (s *Store) AuthRequestByCode(_ context.Context, code string) (op.AuthRequest, error) {
	var out AuthRequest
	err := s.update(func(d *diskState) error {
		for _, a := range d.Auth {
			if a.Code == hash(code) && !a.Consumed && a.Done() && validSession(d.Sessions[a.SessionID], time.Now()) {
				a.Consumed = true
				out = *a
				return nil
			}
		}
		return errDenied
	})
	return &out, err
}
func (s *Store) DeleteAuthRequest(_ context.Context, id string) error {
	return s.update(func(d *diskState) error { delete(d.Auth, id); return nil })
}
func (s *Store) CreateAccessToken(_ context.Context, r op.TokenRequest) (string, time.Time, error) {
	a, ok := r.(*AuthRequest)
	if !ok || !a.Consumed {
		return "", time.Time{}, errDenied
	}
	id := randomToken()
	var expiry time.Time
	err := s.update(func(d *diskState) error {
		v := d.Sessions[a.SessionID]
		if !validSession(v, time.Now()) || time.Until(v.Expires) < 6*time.Minute {
			return errDenied
		}
		expiry = v.Expires
		d.Tokens[id] = &Token{ID: id, SessionID: a.SessionID, ClientID: a.GetClientID(), Expires: expiry}
		return nil
	})
	return id, expiry, err
}
func (s *Store) GetClientByClientID(_ context.Context, id string) (op.Client, error) {
	c := s.clients[id]
	if c == nil {
		return nil, errDenied
	}
	return c, nil
}
func (s *Store) AuthorizeClientIDSecret(_ context.Context, id, secret string) error {
	c := s.clients[id]
	if c == nil || !constantEqual(c.Secret, secret) {
		return errDenied
	}
	return nil
}
func (s *Store) SetUserinfoFromScopes(context.Context, *oidc.UserInfo, string, string, []string) error {
	return nil
}
func (s *Store) SetUserinfoFromRequest(_ context.Context, u *oidc.UserInfo, r op.IDTokenRequest, _ []string) error {
	a, ok := r.(*AuthRequest)
	if !ok {
		return errDenied
	}
	return s.update(func(d *diskState) error {
		if !validSession(d.Sessions[a.SessionID], time.Now()) {
			return errDenied
		}
		u.Subject = d.Owner.ID
		u.Email = s.email
		u.EmailVerified = true
		u.Name = "我的导航"
		u.AppendClaims("sid", hash(a.SessionID))
		return nil
	})
}
func (s *Store) SetUserinfoFromToken(_ context.Context, u *oidc.UserInfo, id, subject, origin string) error {
	return s.update(func(d *diskState) error {
		t := d.Tokens[id]
		if t == nil || subject != d.Owner.ID || !validSession(d.Sessions[t.SessionID], time.Now()) {
			return errDenied
		}
		u.Subject = d.Owner.ID
		u.Email = s.email
		u.EmailVerified = true
		u.AppendClaims("sid", hash(t.SessionID))
		u.AppendClaims("acr", acr)
		return nil
	})
}
func (s *Store) SetIntrospectionFromToken(_ context.Context, u *oidc.IntrospectionResponse, id, subject, client string) error {
	return s.update(func(d *diskState) error {
		t := d.Tokens[id]
		if t == nil || t.ClientID != client || !time.Now().Before(t.Expires) || (subject != "" && subject != d.Owner.ID) {
			return errDenied
		}
		v := d.Sessions[t.SessionID]
		if !validSession(v, time.Now()) {
			return errDenied
		}
		v.LastSeen = time.Now()
		u.Active = true
		u.Subject = d.Owner.ID
		u.ClientID = t.ClientID
		u.Audience = []string{t.ClientID}
		u.Expiration = oidc.FromTime(t.Expires)
		u.AuthTime = oidc.FromTime(v.AuthTime)
		u.Issuer = issuer
		u.Scope = []string{"openid", "email", "profile"}
		u.TokenType = "Bearer"
		u.Claims = map[string]any{"sid": hash(t.SessionID), "acr": acr}
		return nil
	})
}
func (s *Store) CreateAccessAndRefreshTokens(context.Context, op.TokenRequest, string) (string, string, time.Time, error) {
	return "", "", time.Time{}, errDenied
}
func (s *Store) TokenRequestByRefreshToken(context.Context, string) (op.RefreshTokenRequest, error) {
	return nil, op.ErrInvalidRefreshToken
}
func (s *Store) GetRefreshTokenInfo(context.Context, string, string) (string, string, error) {
	return "", "", op.ErrInvalidRefreshToken
}

// The OP logout endpoint is replaced by the browser confirmation route. This
// defense also prevents any library invocation from performing GET logout.
func (s *Store) TerminateSession(context.Context, string, string) error {
	return errors.New("use CSRF-protected portal logout")
}
func (s *Store) RevokeToken(_ context.Context, id, user, client string) *oidc.Error {
	err := s.update(func(d *diskState) error {
		t := d.Tokens[id]
		if t != nil && t.ClientID == client {
			delete(d.Tokens, id)
		}
		return nil
	})
	if err != nil {
		return oidc.ErrServerError()
	}
	return nil
}
func (s *Store) GetPrivateClaimsFromScopes(context.Context, string, string, []string) (map[string]any, error) {
	return nil, nil
}
func (s *Store) GetKeyByIDAndClientID(context.Context, string, string) (*jose.JSONWebKey, error) {
	return nil, errDenied
}
func (s *Store) ValidateJWTProfileScopes(context.Context, string, []string) ([]string, error) {
	return nil, errDenied
}
func (s *Store) Health(ctx context.Context) error { return s.db.PingContext(ctx) }

type providerKey struct {
	key    *rsa.PrivateKey
	public bool
}

func (k providerKey) ID() string                                  { return "nav-rsa-" + hash(k.key.N.String())[:16] }
func (k providerKey) SignatureAlgorithm() jose.SignatureAlgorithm { return jose.RS256 }
func (k providerKey) Algorithm() jose.SignatureAlgorithm          { return jose.RS256 }
func (k providerKey) Use() string                                 { return "sig" }
func (k providerKey) Key() any {
	if k.public {
		return &k.key.PublicKey
	}
	return k.key
}
func (s *Store) SigningKey(context.Context) (op.SigningKey, error) {
	return providerKey{key: s.key}, nil
}
func (s *Store) SignatureAlgorithms(context.Context) ([]jose.SignatureAlgorithm, error) {
	return []jose.SignatureAlgorithm{jose.RS256}, nil
}
func (s *Store) KeySet(context.Context) ([]op.Key, error) {
	return []op.Key{providerKey{key: s.key, public: true}}, nil
}

var _ op.Storage = (*Store)(nil)
