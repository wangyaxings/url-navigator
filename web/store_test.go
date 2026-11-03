package main

import (
	"context"
	"github.com/pquerna/otp/totp"
	"github.com/zitadel/oidc/v3/pkg/oidc"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	s, err := openStore(filepath.Join(t.TempDir(), "portal.db"), [32]byte{1}, "owner@example.test")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.db.Close() })
	return s
}
func TestOwnerPinRejectsSubstitutionAndUnverifiedEmail(t *testing.T) {
	s := testStore(t)
	for _, i := range []Identity{{"a", "other@example.test", true}, {"a", "owner@example.test", false}} {
		if _, err := s.pinOwner(i); err == nil {
			t.Fatal("unauthorized identity accepted")
		}
	}
	id, err := s.pinOwner(Identity{"a", "owner@example.test", true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.pinOwner(Identity{"b", "owner@example.test", true}); err == nil {
		t.Fatal("subject substitution accepted")
	}
	again, err := s.pinOwner(Identity{"a", "owner@example.test", true})
	if err != nil || again != id {
		t.Fatal("unstable subject")
	}
}
func enroll(t *testing.T, s *Store) (string, string, []string) {
	t.Helper()
	_, err := s.pinOwner(Identity{"a", "owner@example.test", true})
	if err != nil {
		t.Fatal(err)
	}
	p, err := s.newPending("")
	if err != nil {
		t.Fatal(err)
	}
	seed, err := s.enrollment(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	code, err := totp.GenerateCode(seed, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	sess, codes, err := s.finishMFA(p.ID, code, "")
	if err != nil {
		t.Fatal(err)
	}
	return sess.ID, seed, codes
}
func TestMFAEncryptionReplayRecoveryConcurrency(t *testing.T) {
	s := testStore(t)
	_, seed, codes := enroll(t, s)
	if len(codes) != 10 {
		t.Fatal("missing recovery codes")
	}
	var raw string
	if err := s.db.QueryRow("SELECT data FROM portal_state WHERE id=1").Scan(&raw); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(raw, seed) || strings.Contains(raw, codes[0]) {
		t.Fatal("MFA secret persisted in plaintext")
	}
	p, _ := s.newPending("")
	code, _ := totp.GenerateCode(seed, time.Now())
	if _, _, err := s.finishMFA(p.ID, code, ""); err == nil {
		t.Fatal("TOTP replay accepted")
	}
	var wins atomic.Int32
	var wg sync.WaitGroup
	for range 12 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p, e := s.newPending("")
			if e != nil {
				return
			}
			if _, _, e = s.finishMFA(p.ID, codes[0], ""); e == nil {
				wins.Add(1)
			}
		}()
	}
	wg.Wait()
	if wins.Load() != 1 {
		t.Fatalf("recovery accepted %d times", wins.Load())
	}
}
func TestIdleAbsoluteAndLogoutRevokeIntrospection(t *testing.T) {
	s := testStore(t)
	sid, _, _ := enroll(t, s)
	if err := s.update(func(d *diskState) error {
		d.Tokens["token"] = &Token{ID: "token", SessionID: sid, ClientID: "pansou", Expires: time.Now().Add(time.Hour)}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	var info oidc.IntrospectionResponse
	if err := s.SetIntrospectionFromToken(context.Background(), &info, "token", "", "pansou"); err != nil || !info.Active {
		t.Fatal("active token rejected", err)
	}
	if err := s.logout(sid); err != nil {
		t.Fatal(err)
	}
	if err := s.SetIntrospectionFromToken(context.Background(), &oidc.IntrospectionResponse{}, "token", "", "pansou"); err == nil {
		t.Fatal("revoked token accepted")
	}
	for _, mode := range []string{"idle", "absolute"} {
		sid, _, _ = enrollRecovery(t, s)
		s.update(func(d *diskState) error {
			if mode == "idle" {
				d.Sessions[sid].LastSeen = time.Now().Add(-61 * time.Minute)
			} else {
				d.Sessions[sid].Expires = time.Now().Add(-time.Second)
			}
			return nil
		})
		if _, err := s.session(sid); err == nil {
			t.Fatal(mode, "expiry bypass")
		}
	}
}
func enrollRecovery(t *testing.T, s *Store) (string, string, []string) {
	t.Helper()
	p, _ := s.newPending("")
	var recovery string
	s.update(func(d *diskState) error { recovery = randomToken(); d.Recovery[hash(recovery)] = true; return nil })
	sess, codes, err := s.finishMFA(p.ID, recovery, "")
	if err != nil {
		t.Fatal(err)
	}
	return sess.ID, "", codes
}
func TestAuthCodeOneUseAndRequiresMFAAndS256(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	req := &oidc.AuthRequest{ClientID: "pansou", RedirectURI: "https://pan.jomon.dev/auth/sso/callback", ResponseType: oidc.ResponseTypeCode, Scopes: []string{"openid", "email", "profile"}, State: "state", Nonce: "nonce"}
	if _, err := s.CreateAuthRequest(ctx, req, ""); err == nil {
		t.Fatal("missing PKCE accepted")
	}
	req.CodeChallenge = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	req.CodeChallengeMethod = oidc.CodeChallengeMethodS256
	a, err := s.CreateAuthRequest(ctx, req, "")
	if err != nil {
		t.Fatal(err)
	}
	if err = s.SaveAuthCode(ctx, a.GetID(), "premature"); err == nil {
		t.Fatal("code before MFA")
	}
	sid, _, _ := enroll(t, s)
	if err = s.completeAuth(a.GetID(), sid); err != nil {
		t.Fatal(err)
	}
	if err = s.SaveAuthCode(ctx, a.GetID(), "code"); err != nil {
		t.Fatal(err)
	}
	var wins atomic.Int32
	var wg sync.WaitGroup
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, e := s.AuthRequestByCode(ctx, "code"); e == nil {
				wins.Add(1)
			}
		}()
	}
	wg.Wait()
	if wins.Load() != 1 {
		t.Fatal("code reused", wins.Load())
	}
}

func TestLogoutAndIntrospectionSerializeAndCannotReviveSession(t *testing.T) {
	s := testStore(t)
	sid, _, _ := enroll(t, s)
	s.update(func(d *diskState) error {
		d.Tokens["race"] = &Token{ID: "race", SessionID: sid, ClientID: "pansou", Expires: time.Now().Add(time.Hour)}
		return nil
	})
	var wg sync.WaitGroup
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 10 {
				s.SetIntrospectionFromToken(context.Background(), &oidc.IntrospectionResponse{}, "race", "", "pansou")
			}
		}()
	}
	if err := s.logout(sid); err != nil {
		t.Fatal(err)
	}
	wg.Wait()
	for range 10 {
		if err := s.SetIntrospectionFromToken(context.Background(), &oidc.IntrospectionResponse{}, "race", "", "pansou"); err == nil {
			t.Fatal("introspection revived logged-out session")
		}
	}
}

func TestMFAFailuresLockAcrossNewPendingSessions(t *testing.T) {
	s := testStore(t)
	_, _, codes := enroll(t, s)
	for range 5 {
		p, _ := s.newPending("")
		if _, _, err := s.finishMFA(p.ID, "incorrect", ""); err == nil {
			t.Fatal("bad MFA accepted")
		}
	}
	p, _ := s.newPending("")
	if _, _, err := s.finishMFA(p.ID, codes[0], ""); err == nil {
		t.Fatal("new pending session bypassed lock")
	}
}

func TestOwnerMFAAndSessionSurviveDatabaseReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "persistent.db")
	s, err := openStore(path, [32]byte{1}, "owner@example.test")
	if err != nil {
		t.Fatal(err)
	}
	sid, _, codes := enroll(t, s)
	id, err := s.pinOwner(Identity{"a", "owner@example.test", true})
	if err != nil {
		t.Fatal(err)
	}
	s.db.Close()
	s, err = openStore(path, [32]byte{1}, "owner@example.test")
	if err != nil {
		t.Fatal(err)
	}
	defer s.db.Close()
	again, err := s.pinOwner(Identity{"a", "owner@example.test", true})
	if err != nil || again != id {
		t.Fatal("owner pin not persisted")
	}
	if _, err = s.session(sid); err != nil {
		t.Fatal("session not persisted", err)
	}
	p, _ := s.newPending("")
	if seed, err := s.enrollment(p.ID); err != nil || seed != "" {
		t.Fatal("MFA enrollment reset")
	}
	if _, _, err = s.finishMFA(p.ID, codes[0], ""); err != nil {
		t.Fatal("recovery not persisted", err)
	}
}
