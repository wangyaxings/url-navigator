package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"image/png"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/pquerna/otp/totp"
	"golang.org/x/oauth2"
)

type fakeGoogle struct {
	identity Identity
	fail     bool
}

func (f fakeGoogle) loginURL(h *Handshake) string {
	return "https://accounts.google.test/authorize?state=" + h.State
}
func (f fakeGoogle) exchange(context.Context, string, *Handshake) (Identity, error) {
	if f.fail {
		return Identity{}, errDenied
	}
	return f.identity, nil
}
func testApp(t *testing.T) (*App, http.Handler) {
	t.Helper()
	s := testStore(t)
	var e error
	s.key, e = rsa.GenerateKey(rand.Reader, 2048)
	if e != nil {
		t.Fatal(e)
	}
	s.clients["pansou"] = &Client{ID: "pansou", Secret: "test-secret-not-real", Redirect: "https://pan.jomon.dev/auth/sso/callback"}
	s.clients["quota"] = &Client{ID: "quota", Secret: "other-test-secret", Redirect: "https://quota.jomon.dev/auth/sso/callback"}
	a, h, e := newApp(s, [32]byte{1}, fakeGoogle{identity: Identity{"a", "owner@example.test", true}}, []Card{{"PanSou", "https://pan.jomon.dev"}, {"Quota", "https://quota.jomon.dev"}})
	if e != nil {
		t.Fatal(e)
	}
	return a, h
}
func request(h http.Handler, method, path string, form url.Values, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	var body string
	if form != nil {
		body = form.Encode()
	}
	r := httptest.NewRequest(method, issuer+path, strings.NewReader(body))
	r.Header.Set("Origin", issuer)
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for _, c := range cookies {
		r.AddCookie(c)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}
func cookie(w *httptest.ResponseRecorder, name string) *http.Cookie {
	for _, c := range w.Result().Cookies() {
		if c.Name == name {
			return c
		}
	}
	return &http.Cookie{Name: name}
}
func TestPendingMFAHasNoPortalLinksAndCSRFEnforced(t *testing.T) {
	a, h := testApp(t)
	w := request(h, "GET", "/", nil)
	if strings.Contains(w.Body.String(), "https://pan.jomon.dev") {
		t.Fatal("anonymous links exposed")
	}
	w = request(h, "GET", "/auth/google/start", nil)
	c := cookie(w, handshakeCookie)
	if !c.Secure || !c.HttpOnly || c.Domain != "" || c.Path != "/" {
		t.Fatal("unsafe cookie")
	}
	var hs *Handshake
	a.store.update(func(d *diskState) error { hs = d.Handshakes[c.Value]; return nil })
	if w = request(h, "GET", "/auth/google/callback?code=valid&state=wrong", nil, c); w.Code != 401 {
		t.Fatal("bad state accepted")
	}
	w = request(h, "GET", "/auth/google/callback?code=valid&state="+hs.State, nil, c)
	pc := cookie(w, pendingCookie)
	if pc.Value == "" {
		t.Fatal("missing pending cookie", w.Code, w.Body.String())
	}
	if w = request(h, "GET", "/auth/google/callback?code=valid&state="+hs.State, nil, c); w.Code != 401 {
		t.Fatal("state replay accepted")
	}
	w = request(h, "GET", "/", nil, pc)
	if strings.Contains(w.Body.String(), "https://pan.jomon.dev") {
		t.Fatal("pending links exposed")
	}
	seed, err := a.store.enrollment(pc.Value)
	if err != nil {
		t.Fatal(err)
	}
	code, _ := totp.GenerateCode(seed, time.Now())
	w = request(h, "POST", "/mfa", url.Values{"code": {code}, "csrf": {"bad"}}, pc)
	if w.Code != 403 {
		t.Fatal("MFA CSRF accepted")
	}
	p, _ := a.store.pending(pc.Value)
	w = request(h, "POST", "/mfa", url.Values{"code": {code}, "csrf": {p.CSRF}}, pc)
	sc := cookie(w, sessionCookie)
	if sc.Value == "" {
		t.Fatal("MFA failed", w.Code, w.Body.String())
	}
	w = request(h, "GET", "/", nil, sc)
	if !strings.Contains(w.Body.String(), "https://pan.jomon.dev") {
		t.Fatal("authenticated links absent")
	}
	w = request(h, "GET", "/logout", nil, sc)
	if _, err = a.store.session(sc.Value); err != nil {
		t.Fatal("GET logout mutated")
	}
	w = request(h, "POST", "/logout", url.Values{"csrf": {"bad"}}, sc)
	if w.Code != 403 {
		t.Fatal("logout CSRF accepted")
	}
	sess, _ := a.store.session(sc.Value)
	w = request(h, "POST", "/logout", url.Values{"csrf": {sess.CSRF}}, sc)
	if _, err = a.store.session(sc.Value); err == nil {
		t.Fatal("logout not revoked")
	}
}
func TestProviderCodePKCEClaimsIntrospectionLogout(t *testing.T) {
	a, h := testApp(t)
	sid, _, _ := enroll(t, a.store)
	sc := &http.Cookie{Name: sessionCookie, Value: sid}
	verifier := oauth2.GenerateVerifier()
	q := url.Values{"client_id": {"pansou"}, "redirect_uri": {"https://pan.jomon.dev/auth/sso/callback"}, "response_type": {"code"}, "scope": {"openid email profile"}, "state": {"client-state"}, "nonce": {"client-nonce"}, "code_challenge": {oauth2.S256ChallengeFromVerifier(verifier)}, "code_challenge_method": {"S256"}}
	w := request(h, "GET", "/authorize?"+q.Encode(), nil, sc)
	if w.Code != 302 {
		t.Fatal("authorize", w.Code, w.Body.String())
	}
	u, _ := url.Parse(w.Header().Get("Location"))
	w = request(h, "GET", u.RequestURI(), nil, sc)
	if w.Code != 302 {
		t.Fatal("login", w.Code, w.Body.String())
	}
	u, _ = url.Parse(w.Header().Get("Location"))
	w = request(h, "GET", u.RequestURI(), nil, sc)
	u, _ = url.Parse(w.Header().Get("Location"))
	code := u.Query().Get("code")
	if code == "" {
		t.Fatal("no code", w.Code, w.Body.String(), u)
	}
	form := url.Values{"grant_type": {"authorization_code"}, "code": {code}, "redirect_uri": {"https://pan.jomon.dev/auth/sso/callback"}, "code_verifier": {verifier}}
	call := func(path string, values url.Values, secret string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", issuer+path, strings.NewReader(values.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		r.SetBasicAuth("pansou", secret)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	w = call("/oauth/token", form, "test-secret-not-real")
	if w.Code != 200 {
		t.Fatal("token", w.Code, w.Body.String())
	}
	var tokens map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &tokens); err != nil {
		t.Fatal(err)
	}
	if tokens["id_token"] == nil || tokens["access_token"] == nil {
		t.Fatal("missing tokens")
	}
	parsed, err := jwt.ParseSigned(tokens["id_token"].(string), []jose.SignatureAlgorithm{jose.RS256})
	if err != nil {
		t.Fatal(err)
	}
	var idClaims map[string]any
	if err = parsed.Claims(&a.store.key.PublicKey, &idClaims); err != nil {
		t.Fatal("signature", err)
	}
	if idClaims["iss"] != issuer || idClaims["nonce"] != "client-nonce" || idClaims["acr"] != acr || idClaims["sid"] != hash(sid) || idClaims["email_verified"] != true || idClaims["auth_time"] == nil {
		t.Fatal("ID claims", idClaims)
	}
	sess, _ := a.store.session(sid)
	if int64(idClaims["exp"].(float64)) > sess.Expires.Unix() {
		t.Fatal("ID token outlives parent")
	}
	if w = call("/oauth/token", form, "test-secret-not-real"); w.Code == 200 {
		t.Fatal("code replay")
	}
	f := url.Values{"token": {tokens["access_token"].(string)}}
	if w = call("/oauth/introspect", f, "bad"); w.Code == 200 {
		t.Fatal("unauthenticated introspection")
	}
	w = call("/oauth/introspect", f, "test-secret-not-real")
	var claims map[string]any
	json.Unmarshal(w.Body.Bytes(), &claims)
	if claims["active"] != true || claims["sid"] != hash(sid) || claims["acr"] != acr || claims["client_id"] != "pansou" {
		t.Fatal("introspection", w.Code, w.Body.String())
	}
	if _, err := a.store.session(claims["sid"].(string)); err == nil {
		t.Fatal("public sid is a portal bearer credential")
	}
	a.store.logout(sid)
	w = call("/oauth/introspect", f, "test-secret-not-real")
	json.Unmarshal(w.Body.Bytes(), &claims)
	if claims["active"] != false {
		t.Fatal("revoked introspection", w.Body.String())
	}
}

func TestDirectHostForgedHeadersAndCrossOriginMutationDenied(t *testing.T) {
	a, h := testApp(t)
	sid, _, _ := enroll(t, a.store)
	sess, _ := a.store.session(sid)
	r := httptest.NewRequest("GET", "http://127.0.0.1/", nil)
	r.Header.Set("X-Forwarded-Host", "nav.jomon.dev")
	r.Header.Set("X-Auth-User", "owner")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 404 {
		t.Fatal("direct host accepted")
	}
	r = httptest.NewRequest("GET", issuer+"/", nil)
	r.Header.Set("X-Auth-User", "owner")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if strings.Contains(w.Body.String(), "https://pan.jomon.dev") {
		t.Fatal("identity header trusted")
	}
	r = httptest.NewRequest("POST", issuer+"/logout", strings.NewReader(url.Values{"csrf": {sess.CSRF}}.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("Origin", "https://evil.test")
	r.AddCookie(&http.Cookie{Name: sessionCookie, Value: sid})
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal("cross-origin logout accepted")
	}
}

func TestMFACompletesForcedLoginWithoutRestartingGoogle(t *testing.T) {
	a, h := testApp(t)
	_, _, codes := enroll(t, a.store)
	q := url.Values{"client_id": {"pansou"}, "redirect_uri": {"https://pan.jomon.dev/auth/sso/callback"}, "response_type": {"code"}, "scope": {"openid email profile"}, "state": {"state"}, "nonce": {"nonce"}, "prompt": {"login"}, "code_challenge": {oauth2.S256ChallengeFromVerifier(oauth2.GenerateVerifier())}, "code_challenge_method": {"S256"}}
	w := request(h, "GET", "/authorize?"+q.Encode(), nil)
	u, _ := url.Parse(w.Header().Get("Location"))
	id := u.Query().Get("request")
	p, err := a.store.newPending(id)
	if err != nil {
		t.Fatal(err)
	}
	w = request(h, "POST", "/mfa", url.Values{"csrf": {p.CSRF}, "code": {codes[0]}}, &http.Cookie{Name: pendingCookie, Value: p.ID})
	sc := cookie(w, sessionCookie)
	u, _ = url.Parse(w.Header().Get("Location"))
	w = request(h, "GET", u.RequestURI(), nil, sc)
	u, _ = url.Parse(w.Header().Get("Location"))
	if u.Query().Get("code") == "" {
		t.Fatal("fresh MFA should complete forced login", w.Code, w.Body.String())
	}
}

func TestEnrollmentQRRequiresPendingIdentity(t *testing.T) {
	a, h := testApp(t)
	if w := request(h, "GET", "/mfa/qr", nil); w.Code != 401 {
		t.Fatal("anonymous QR access")
	}
	a.store.pinOwner(Identity{"a", "owner@example.test", true})
	p, _ := a.store.newPending("")
	pc := &http.Cookie{Name: pendingCookie, Value: p.ID}
	w := request(h, "GET", "/mfa/qr", nil, pc)
	if w.Code != 200 || w.Header().Get("Content-Type") != "image/png" {
		t.Fatal("QR unavailable", w.Code)
	}
	img, err := png.Decode(w.Body)
	if err != nil || img.Bounds().Dx() != 240 {
		t.Fatal("invalid QR", err)
	}
}

func TestProviderRejectsUnregisteredRedirectAndIncompleteProof(t *testing.T) {
	a, h := testApp(t)
	for _, field := range []string{"redirect_uri", "nonce", "state", "code_challenge", "code_challenge_method"} {
		t.Run(field, func(t *testing.T) {
			q := url.Values{"client_id": {"pansou"}, "redirect_uri": {"https://pan.jomon.dev/auth/sso/callback"}, "response_type": {"code"}, "scope": {"openid email profile"}, "state": {"state"}, "nonce": {"nonce"}, "code_challenge": {oauth2.S256ChallengeFromVerifier(oauth2.GenerateVerifier())}, "code_challenge_method": {"S256"}}
			if field == "redirect_uri" {
				q.Set(field, "https://evil.test/callback")
			} else if field == "code_challenge_method" {
				q.Set(field, "plain")
			} else {
				q.Del(field)
			}
			w := request(h, "GET", "/authorize?"+q.Encode(), nil)
			if strings.HasPrefix(w.Header().Get("Location"), "https://evil.test") {
				t.Fatal("unregistered redirect followed")
			}
			a.store.update(func(d *diskState) error {
				if len(d.Auth) != 0 {
					t.Fatal("incomplete authorization proof persisted")
				}
				return nil
			})
		})
	}
}

func TestExpiredAuthorizationStillShowsEnrollmentRecoveryCodes(t *testing.T) {
	a, h := testApp(t)
	a.store.pinOwner(Identity{"a", "owner@example.test", true})
	p, _ := a.store.newPending("expired-request")
	seed, _ := a.store.enrollment(p.ID)
	code, _ := totp.GenerateCode(seed, time.Now())
	w := request(h, "POST", "/mfa", url.Values{"csrf": {p.CSRF}, "code": {code}}, &http.Cookie{Name: pendingCookie, Value: p.ID})
	if w.Code != 200 || !strings.Contains(w.Body.String(), "保存恢复码") {
		t.Fatal("expired authorization lost first-enrollment recovery codes")
	}
}
