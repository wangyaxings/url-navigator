package main

import (
	"context"
	_ "embed"
	"html/template"
	"image/png"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pquerna/otp"
	"github.com/zitadel/oidc/v3/pkg/op"
)

const sessionCookie = "__Host-nav_session"
const pendingCookie = "__Host-nav_pending"
const handshakeCookie = "__Host-nav_oauth"

type Card struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}
type App struct {
	store    *Store
	google   upstream
	cards    []Card
	callback func(context.Context, string) string
}

//go:embed page.html
var pageHTML string
var page = template.Must(template.New("page").Parse(pageHTML))

type pageData struct {
	Title, View, Error, CSRF, Seed, AuthID, Continue string
	Cards                                            []Card
	Codes                                            []string
}

func newApp(s *Store, master [32]byte, g upstream, cards []Card) (*App, http.Handler, error) {
	provider, err := op.NewProvider(&op.Config{CryptoKey: master, CryptoKeyId: "nav-encryption-v1", CodeMethodS256: true, DefaultLogoutRedirectURI: issuer + "/logout"}, s, op.StaticIssuer(issuer), op.WithCustomEndSessionEndpoint(op.NewEndpoint("logout")))
	if err != nil {
		return nil, nil, err
	}
	a := &App{store: s, google: g, cards: cards, callback: op.AuthCallbackURL(provider)}
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			a.home(w, r)
		} else {
			provider.ServeHTTP(w, r)
		}
	})
	mux.HandleFunc("/login", a.login)
	mux.HandleFunc("/auth/google/start", a.googleStart)
	mux.HandleFunc("/auth/google/callback", a.googleCallback)
	mux.HandleFunc("/mfa", a.mfa)
	mux.HandleFunc("/logout", a.logout)
	mux.HandleFunc("/mfa/qr", a.qr)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(405)
			return
		}
		if err := s.Health(r.Context()); err != nil {
			w.WriteHeader(503)
			return
		}
		w.Write([]byte("ok"))
	})
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Pragma", "no-cache")
		// Native form POSTs need their same-origin Origin for the strict CSRF
		// check; no-referrer would make browsers serialize it as null.
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		// The MFA form's 303 -> OIDC callback -> client callback chain is a
		// form navigation. Permit only the two fixed registered client origins.
		w.Header().Set("Content-Security-Policy", "default-src 'none'; img-src 'self'; style-src 'unsafe-inline'; form-action 'self' https://pan.jomon.dev https://quota.jomon.dev; base-uri 'none'; frame-ancestors 'none'")
		if r.Host != "nav.jomon.dev" {
			http.Error(w, "Not found", 404)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
		mux.ServeHTTP(w, r)
	})
	return a, h, nil
}
func render(w http.ResponseWriter, status int, d pageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_ = page.Execute(w, d)
}
func failure(w http.ResponseWriter, status int) {
	render(w, status, pageData{Title: "无法完成验证", View: "error", Error: "验证未完成，请稍后重新登录。"})
}
func only(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method != method {
		w.Header().Set("Allow", method)
		w.WriteHeader(405)
		return false
	}
	return true
}
func cookieValue(r *http.Request, name string) string {
	c, err := r.Cookie(name)
	if err != nil {
		return ""
	}
	return c.Value
}
func setCookie(w http.ResponseWriter, name, value string, seconds int) {
	http.SetCookie(w, &http.Cookie{Name: name, Value: value, Path: "/", Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: seconds})
}
func csrf(r *http.Request, want string) bool {
	return r.Header.Get("Origin") == issuer && want != "" && constantEqual(r.FormValue("csrf"), want)
}
func (a *App) home(w http.ResponseWriter, r *http.Request) {
	if !only(w, r, "GET") {
		return
	}
	v, err := a.store.session(cookieValue(r, sessionCookie))
	if err != nil {
		render(w, 200, pageData{Title: "我的导航", View: "login"})
		return
	}
	render(w, 200, pageData{Title: "我的导航", View: "home", CSRF: v.CSRF, Cards: a.cards})
}
func (a *App) login(w http.ResponseWriter, r *http.Request) {
	if !only(w, r, "GET") {
		return
	}
	id := r.URL.Query().Get("request")
	if id == "" {
		http.Redirect(w, r, "/", 303)
		return
	}
	auth, err := a.store.AuthRequestByID(r.Context(), id)
	if err != nil {
		failure(w, 401)
		return
	}
	v, err := a.store.session(cookieValue(r, sessionCookie))
	if err == nil {
		force := false
		for _, prompt := range auth.(*AuthRequest).Request.Prompt {
			if prompt == "login" || prompt == "select_account" {
				force = true
			}
		}
		if !force && a.store.completeAuth(id, v.ID) == nil {
			http.Redirect(w, r, a.callback(op.ContextWithIssuer(r.Context(), issuer), id), 302)
			return
		}
	}
	if strings.Contains(strings.Join(auth.(*AuthRequest).Request.Prompt, " "), "none") {
		failure(w, 401)
		return
	}
	render(w, 200, pageData{Title: "登录我的导航", View: "login", AuthID: id})
}
func (a *App) googleStart(w http.ResponseWriter, r *http.Request) {
	if !only(w, r, "GET") {
		return
	}
	id := r.URL.Query().Get("request")
	if id != "" {
		if _, err := a.store.AuthRequestByID(r.Context(), id); err != nil {
			failure(w, 401)
			return
		}
	}
	h, err := a.store.newHandshake(id)
	if err != nil {
		failure(w, 429)
		return
	}
	setCookie(w, handshakeCookie, h.ID, 300)
	http.Redirect(w, r, a.google.loginURL(h), 302)
}
func (a *App) googleCallback(w http.ResponseWriter, r *http.Request) {
	if !only(w, r, "GET") {
		return
	}
	h, err := a.store.consumeHandshake(cookieValue(r, handshakeCookie), r.URL.Query().Get("state"))
	if err != nil {
		failure(w, 401)
		return
	}
	setCookie(w, handshakeCookie, "", -1)
	if r.URL.Query().Get("error") != "" || r.URL.Query().Get("code") == "" {
		failure(w, 401)
		return
	}
	i, err := a.google.exchange(r.Context(), r.URL.Query().Get("code"), h)
	if err != nil {
		failure(w, 401)
		return
	}
	if _, err = a.store.pinOwner(i); err != nil {
		failure(w, 401)
		return
	}
	p, err := a.store.newPending(h.AuthID)
	if err != nil {
		failure(w, 401)
		return
	}
	setCookie(w, pendingCookie, p.ID, 300)
	http.Redirect(w, r, "/mfa", 303)
}
func (a *App) mfa(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" && r.Method != "POST" {
		w.WriteHeader(405)
		return
	}
	p, err := a.store.pending(cookieValue(r, pendingCookie))
	if err != nil {
		failure(w, 401)
		return
	}
	if r.Method == "GET" {
		seed, err := a.store.enrollment(p.ID)
		if err != nil {
			failure(w, 401)
			return
		}
		render(w, 200, pageData{Title: "安全验证", View: "mfa", CSRF: p.CSRF, Seed: seed})
		return
	}
	if !csrf(r, p.CSRF) {
		failure(w, 403)
		return
	}
	v, codes, err := a.store.finishMFA(p.ID, strings.TrimSpace(r.FormValue("code")))
	if err != nil {
		render(w, 401, pageData{Title: "安全验证", View: "mfa", CSRF: p.CSRF, Error: "验证失败，请检查后重试；多次失败后请等待五分钟。"})
		return
	}
	setCookie(w, pendingCookie, "", -1)
	setCookie(w, sessionCookie, v.ID, int((12 * time.Hour).Seconds()))
	next := "/"
	if p.AuthID != "" {
		if err = a.store.completeAuth(p.AuthID, v.ID); err == nil {
			next = a.callback(op.ContextWithIssuer(r.Context(), issuer), p.AuthID)
		}
	}
	if len(codes) > 0 {
		render(w, 200, pageData{Title: "保存恢复码", View: "recovery", Codes: codes, Continue: next})
		return
	}
	http.Redirect(w, r, next, 303)
}
func (a *App) logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" && r.Method != "POST" {
		w.WriteHeader(405)
		return
	}
	v, err := a.store.session(cookieValue(r, sessionCookie))
	if err != nil {
		if r.Method == "POST" {
			failure(w, 403)
		} else {
			render(w, 200, pageData{Title: "已退出", View: "loggedout"})
		}
		return
	}
	if r.Method == "GET" {
		render(w, 200, pageData{Title: "退出登录", View: "logout", CSRF: v.CSRF})
		return
	}
	if !csrf(r, v.CSRF) {
		failure(w, 403)
		return
	}
	if err = a.store.logout(v.ID); err != nil {
		failure(w, 503)
		return
	}
	setCookie(w, sessionCookie, "", -1)
	setCookie(w, pendingCookie, "", -1)
	render(w, 200, pageData{Title: "已退出", View: "loggedout"})
}
func (a *App) qr(w http.ResponseWriter, r *http.Request) {
	if !only(w, r, "GET") {
		return
	}
	seed, err := a.store.enrollment(cookieValue(r, pendingCookie))
	if err != nil || seed == "" {
		failure(w, 401)
		return
	}
	key, err := otp.NewKeyFromURL("otpauth://totp/Jomon:owner?issuer=Jomon&secret=" + url.QueryEscape(seed))
	if err != nil {
		failure(w, 500)
		return
	}
	img, err := key.Image(240, 240)
	if err != nil {
		failure(w, 500)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	_ = png.Encode(w, img)
}
