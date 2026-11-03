package main

import (
	"context"
	"encoding/json"
	"github.com/zitadel/oidc/v3/pkg/oidc"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"golang.org/x/oauth2"
)

func TestFirstMFAWithoutPreviousCookiePreservesPendingAuthorization(t *testing.T) {
	a, _ := testApp(t)
	auth, err := a.store.CreateAuthRequest(context.Background(), &oidc.AuthRequest{ClientID: "pansou", RedirectURI: a.store.clients["pansou"].Redirect, ResponseType: oidc.ResponseTypeCode, Scopes: []string{"openid", "email", "profile"}, State: "state", Nonce: "nonce", CodeChallenge: oauth2.S256ChallengeFromVerifier(oauth2.GenerateVerifier()), CodeChallengeMethod: oidc.CodeChallengeMethodS256}, "")
	if err != nil {
		t.Fatal(err)
	}
	enroll(t, a.store)
	if _, err = a.store.AuthRequestByID(context.Background(), auth.GetID()); err != nil {
		t.Fatal("initial MFA removed pending application authorization", err)
	}
}

func providerPost(h http.Handler, path, client, secret string, form url.Values) *httptest.ResponseRecorder {
	r := httptest.NewRequest("POST", issuer+path, strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.SetBasicAuth(client, secret)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func clientCode(t *testing.T, a *App, h http.Handler, sid, client string) url.Values {
	t.Helper()
	verifier := oauth2.GenerateVerifier()
	c := a.store.clients[client]
	q := url.Values{"client_id": {client}, "redirect_uri": {c.Redirect}, "response_type": {"code"}, "scope": {"openid email profile"}, "state": {"state"}, "nonce": {"nonce"}, "code_challenge": {oauth2.S256ChallengeFromVerifier(verifier)}, "code_challenge_method": {"S256"}}
	w := request(h, "GET", "/authorize?"+q.Encode(), nil, &http.Cookie{Name: sessionCookie, Value: sid})
	for range 2 {
		u, err := url.Parse(w.Header().Get("Location"))
		if err != nil || w.Code != 302 {
			t.Fatal("authorization flow", w.Code, err)
		}
		w = request(h, "GET", u.RequestURI(), nil, &http.Cookie{Name: sessionCookie, Value: sid})
	}
	u, err := url.Parse(w.Header().Get("Location"))
	if err != nil || u.Query().Get("code") == "" {
		t.Fatal("missing authorization code", w.Code, err)
	}
	return url.Values{"grant_type": {"authorization_code"}, "code": {u.Query().Get("code")}, "redirect_uri": {c.Redirect}, "code_verifier": {verifier}}
}

func accessToken(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		AccessToken string `json:"access_token"`
	}
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &body) != nil || body.AccessToken == "" {
		t.Fatal("token response", w.Code, w.Body.String())
	}
	return body.AccessToken
}
func introspectionActive(t *testing.T, a *App, h http.Handler, client, token string) bool {
	t.Helper()
	c := a.store.clients[client]
	w := providerPost(h, "/oauth/introspect", client, c.Secret, url.Values{"token": {token}})
	var response struct {
		Active bool `json:"active"`
	}
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &response) != nil {
		t.Fatal("introspection failed", w.Code, w.Body.String())
	}
	return response.Active
}

func TestMFAReplacementRetiresBothApplicationsAndLogoutCannotLeaveOldParent(t *testing.T) {
	a, h := testApp(t)
	sid, _, codes := enroll(t, a.store)
	old := &http.Cookie{Name: sessionCookie, Value: sid}
	tokens := map[string]string{}
	for _, client := range []string{"pansou", "quota"} {
		tokens[client] = accessToken(t, providerPost(h, "/oauth/token", client, a.store.clients[client].Secret, clientCode(t, a, h, sid, client)))
		if !introspectionActive(t, a, h, client, tokens[client]) {
			t.Fatal("initial token inactive")
		}
	}
	p, _ := a.store.newPending("")
	pc := &http.Cookie{Name: pendingCookie, Value: p.ID}
	// A failed challenge must leave the existing parent and its tokens usable.
	w := request(h, "POST", "/mfa", url.Values{"csrf": {p.CSRF}, "code": {"incorrect"}}, pc, old)
	if w.Code != 401 {
		t.Fatal("invalid MFA accepted")
	}
	for client, token := range tokens {
		if !introspectionActive(t, a, h, client, token) {
			t.Fatal("failed MFA retired active parent")
		}
	}
	w = request(h, "POST", "/mfa", url.Values{"csrf": {p.CSRF}, "code": {codes[0]}}, pc, old)
	replacement := cookie(w, sessionCookie)
	if replacement.Value == "" || replacement.Value == sid {
		t.Fatal("missing replacement session")
	}
	for client, token := range tokens {
		if introspectionActive(t, a, h, client, token) {
			t.Fatal("replaced parent still authorizes", client)
		}
	}
	sess, err := a.store.session(replacement.Value)
	if err != nil {
		t.Fatal(err)
	}
	w = request(h, "POST", "/logout", url.Values{"csrf": {sess.CSRF}}, replacement)
	if w.Code != 200 {
		t.Fatal("logout failed", w.Code)
	}
	for client, token := range tokens {
		if introspectionActive(t, a, h, client, token) {
			t.Fatal("logout left old application authorized", client)
		}
	}
}

func TestMFAReplacementRacingOldCodeIssuanceCannotRetainToken(t *testing.T) {
	a, h := testApp(t)
	sid, _, codes := enroll(t, a.store)
	form := clientCode(t, a, h, sid, "pansou")
	p, _ := a.store.newPending("")
	var issued, replaced *httptest.ResponseRecorder
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		issued = providerPost(h, "/oauth/token", "pansou", a.store.clients["pansou"].Secret, form)
	}()
	go func() {
		defer wg.Done()
		<-start
		replaced = request(h, "POST", "/mfa", url.Values{"csrf": {p.CSRF}, "code": {codes[0]}}, &http.Cookie{Name: pendingCookie, Value: p.ID}, &http.Cookie{Name: sessionCookie, Value: sid})
	}()
	close(start)
	wg.Wait()
	if cookie(replaced, sessionCookie).Value == "" {
		t.Fatal("replacement failed")
	}
	if issued.Code == 200 && introspectionActive(t, a, h, "pansou", accessToken(t, issued)) {
		t.Fatal("old code minted a surviving token")
	}
	if w := providerPost(h, "/oauth/token", "pansou", a.store.clients["pansou"].Secret, form); w.Code == 200 {
		t.Fatal("retired parent code accepted after replacement")
	}
	if _, err := a.store.session(sid); err == nil {
		t.Fatal("old browser parent survived replacement")
	}
}

func TestConsumedOldCodeCannotIssueAfterMFAReplacement(t *testing.T) {
	a, h := testApp(t)
	sid, _, codes := enroll(t, a.store)
	form := clientCode(t, a, h, sid, "pansou")
	auth, err := a.store.AuthRequestByCode(context.Background(), form.Get("code"))
	if err != nil {
		t.Fatal(err)
	}
	p, _ := a.store.newPending("")
	w := request(h, "POST", "/mfa", url.Values{"csrf": {p.CSRF}, "code": {codes[0]}}, &http.Cookie{Name: pendingCookie, Value: p.ID}, &http.Cookie{Name: sessionCookie, Value: sid})
	if cookie(w, sessionCookie).Value == "" {
		t.Fatal("replacement failed")
	}
	if _, _, err = a.store.CreateAccessToken(context.Background(), auth); err == nil {
		t.Fatal("in-flight consumed code resurrected retired parent")
	}
}

func TestConcurrentMFAReplacementsCannotForkBrowserParents(t *testing.T) {
	a, h := testApp(t)
	sid, _, codes := enroll(t, a.store)
	p1, _ := a.store.newPending("")
	p2, _ := a.store.newPending("")
	var responses [2]*httptest.ResponseRecorder
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i, p := range []*Pending{p1, p2} {
		wg.Add(1)
		go func(i int, p *Pending) {
			defer wg.Done()
			<-start
			responses[i] = request(h, "POST", "/mfa", url.Values{"csrf": {p.CSRF}, "code": {codes[i]}}, &http.Cookie{Name: pendingCookie, Value: p.ID}, &http.Cookie{Name: sessionCookie, Value: sid})
		}(i, p)
	}
	close(start)
	wg.Wait()
	wins := 0
	for _, w := range responses {
		if cookie(w, sessionCookie).Value != "" {
			wins++
		}
	}
	if wins != 1 {
		t.Fatalf("same browser parent replaced %d times", wins)
	}
}

func TestProviderTransportRejectsGETQueryAndFormCredentials(t *testing.T) {
	a, h := testApp(t)
	sid, _, _ := enroll(t, a.store)
	for _, endpoint := range []string{"/oauth/token", "/oauth/introspect", "/revoke"} {
		t.Run(endpoint, func(t *testing.T) {
			form := clientCode(t, a, h, sid, "pansou")
			token := accessToken(t, providerPost(h, "/oauth/token", "pansou", a.store.clients["pansou"].Secret, form))
			for _, test := range []struct {
				name, method, query      string
				bodyCredentials, noBasic bool
				want                     int
			}{
				{name: "get", method: "GET", want: 405},
				{name: "query secret", method: "POST", query: "client_secret=private", want: 400},
				{name: "query bearer", method: "POST", query: "token=private", want: 400},
				{name: "query grant", method: "POST", query: "grant_type=authorization_code", want: 400},
				{name: "form secret", method: "POST", bodyCredentials: true, noBasic: true, want: 400},
				{name: "form secret with Basic", method: "POST", bodyCredentials: true, want: 400},
				{name: "missing Basic", method: "POST", noBasic: true, want: 401},
			} {
				t.Run(test.name, func(t *testing.T) {
					body := url.Values{"token": {token}}
					if endpoint == "/oauth/token" {
						body = clientCode(t, a, h, sid, "pansou")
					}
					if test.bodyCredentials {
						body.Set("client_id", "pansou")
						body.Set("client_secret", a.store.clients["pansou"].Secret)
					}
					path := issuer + endpoint
					if test.query != "" {
						path += "?" + test.query
					}
					if test.method == "GET" {
						path += "?" + body.Encode()
					}
					r := httptest.NewRequest(test.method, path, strings.NewReader(body.Encode()))
					r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
					if !test.noBasic {
						r.SetBasicAuth("pansou", a.store.clients["pansou"].Secret)
					}
					w := httptest.NewRecorder()
					h.ServeHTTP(w, r)
					if w.Code != test.want {
						t.Fatalf("transport accepted or misclassified: %d, want %d", w.Code, test.want)
					}
					if test.method == "GET" && w.Header().Get("Allow") != "POST" {
						t.Fatal("missing method boundary")
					}
				})
			}
			if !introspectionActive(t, a, h, "pansou", token) {
				t.Fatal("rejected transport mutated token")
			}
			if endpoint == "/revoke" {
				w := providerPost(h, endpoint, "pansou", a.store.clients["pansou"].Secret, url.Values{"token": {token}})
				if w.Code != 200 || introspectionActive(t, a, h, "pansou", token) {
					t.Fatal("valid Basic revocation failed", w.Code)
				}
			}
		})
	}
}

func TestProviderTransportRejectsAmbiguousAndNonFormRequests(t *testing.T) {
	a, h := testApp(t)
	for _, endpoint := range []string{"/oauth/token", "/oauth/introspect", "/revoke"} {
		for _, test := range []struct {
			name, contentType, body string
			duplicateAuth           bool
			want                    int
		}{
			{"JSON", "application/json", `{"token":"value"}`, false, 415},
			{"malformed form", "application/x-www-form-urlencoded", "token=%zz", false, 400},
			{"duplicate token", "application/x-www-form-urlencoded", "token=one&token=two", false, 400},
			{"client assertion", "application/x-www-form-urlencoded", "client_assertion=value", false, 400},
			{"duplicate authorization", "application/x-www-form-urlencoded", "token=value", true, 401},
		} {
			t.Run(endpoint+"/"+test.name, func(t *testing.T) {
				r := httptest.NewRequest("POST", issuer+endpoint, strings.NewReader(test.body))
				r.Header.Set("Content-Type", test.contentType)
				r.SetBasicAuth("pansou", a.store.clients["pansou"].Secret)
				if test.duplicateAuth {
					r.Header.Add("Authorization", r.Header.Get("Authorization"))
				}
				w := httptest.NewRecorder()
				h.ServeHTTP(w, r)
				if w.Code != test.want {
					t.Fatalf("ambiguous transport accepted: %d, want %d", w.Code, test.want)
				}
			})
		}
	}
}
