package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pquerna/otp/totp"
	_ "modernc.org/sqlite"
)

var errDenied = errors.New("authentication failed")

const acr = "urn:jomon:acr:google-totp"
const issuer = "https://nav.jomon.dev"

type Identity struct {
	Subject, Email string
	Verified       bool
}
type Owner struct {
	ID, GoogleSub, Seed string
	LastStep            int64
	Failures            int
	LockedUntil         time.Time
}
type Session struct {
	ID, CSRF                    string
	AuthTime, LastSeen, Expires time.Time
}
type Pending struct {
	ID, CSRF, AuthID, Seed string
	Expires                time.Time
}
type Handshake struct {
	ID, State, Nonce, Verifier, AuthID string
	Expires                            time.Time
}
type Token struct {
	ID, SessionID, ClientID string
	Expires                 time.Time
}
type diskState struct {
	Owner      Owner
	Recovery   map[string]bool
	Sessions   map[string]*Session
	Pending    map[string]*Pending
	Handshakes map[string]*Handshake
	Auth       map[string]*AuthRequest
	Tokens     map[string]*Token
}

// A single SQLite row is a transactionally updated aggregate for this single-owner
// portal. All consumers read within the same transaction: logout, token issuance,
// recovery redemption and introspection cannot see partially updated state.
type Store struct {
	db      *sql.DB
	mu      sync.Mutex
	aead    cipher.AEAD
	email   string
	clients map[string]*Client
	key     *rsa.PrivateKey
}

func openStore(path string, master [32]byte, email string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	fail := func(e error) (*Store, error) { db.Close(); return nil, e }
	if _, err = db.Exec("PRAGMA journal_mode=WAL; PRAGMA busy_timeout=5000; CREATE TABLE IF NOT EXISTS portal_state (id INTEGER PRIMARY KEY CHECK(id=1), data TEXT NOT NULL)"); err != nil {
		return fail(err)
	}
	if _, err = db.Exec("INSERT OR IGNORE INTO portal_state(id,data) VALUES(1,'{}')"); err != nil {
		return fail(err)
	}
	if err = os.Chmod(path, 0600); err != nil {
		return fail(err)
	}
	block, err := aes.NewCipher(master[:])
	if err != nil {
		return fail(err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return fail(err)
	}
	return &Store{db: db, aead: aead, email: email, clients: map[string]*Client{}}, nil
}

func (s *Store) update(fn func(*diskState) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var raw []byte
	if err = tx.QueryRow("SELECT data FROM portal_state WHERE id=1").Scan(&raw); err != nil {
		return err
	}
	var d diskState
	if err = json.Unmarshal(raw, &d); err != nil {
		return err
	}
	if d.Recovery == nil {
		d.Recovery = map[string]bool{}
	}
	if d.Sessions == nil {
		d.Sessions = map[string]*Session{}
	}
	if d.Pending == nil {
		d.Pending = map[string]*Pending{}
	}
	if d.Handshakes == nil {
		d.Handshakes = map[string]*Handshake{}
	}
	if d.Auth == nil {
		d.Auth = map[string]*AuthRequest{}
	}
	if d.Tokens == nil {
		d.Tokens = map[string]*Token{}
	}
	now := time.Now()
	for k, v := range d.Pending {
		if !now.Before(v.Expires) {
			delete(d.Pending, k)
		}
	}
	for k, v := range d.Handshakes {
		if !now.Before(v.Expires) {
			delete(d.Handshakes, k)
		}
	}
	for k, v := range d.Auth {
		if !now.Before(v.Expires) {
			delete(d.Auth, k)
		}
	}
	for k, v := range d.Sessions {
		if !validSession(v, now) {
			delete(d.Sessions, k)
		}
	}
	for k, v := range d.Tokens {
		if !now.Before(v.Expires) || d.Sessions[v.SessionID] == nil {
			delete(d.Tokens, k)
		}
	}
	if err = fn(&d); err != nil {
		return err
	}
	raw, err = json.Marshal(d)
	if err != nil {
		return err
	}
	if _, err = tx.Exec("UPDATE portal_state SET data=? WHERE id=1", raw); err != nil {
		return err
	}
	return tx.Commit()
}
func randomToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("secure randomness unavailable")
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
func hash(v string) string { h := sha256.Sum256([]byte(v)); return hex.EncodeToString(h[:]) }
func (s *Store) seal(v string) string {
	nonce := make([]byte, s.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		panic("secure randomness unavailable")
	}
	return base64.RawStdEncoding.EncodeToString(s.aead.Seal(nonce, nonce, []byte(v), []byte("nav-totp-v1")))
}
func (s *Store) unseal(v string) (string, error) {
	b, err := base64.RawStdEncoding.DecodeString(v)
	if err != nil || len(b) < s.aead.NonceSize() {
		return "", errDenied
	}
	n := s.aead.NonceSize()
	b, err = s.aead.Open(nil, b[:n], b[n:], []byte("nav-totp-v1"))
	return string(b), err
}
func (s *Store) pinOwner(i Identity) (string, error) {
	if !i.Verified || i.Email != s.email || i.Subject == "" {
		return "", errDenied
	}
	var id string
	err := s.update(func(d *diskState) error {
		if d.Owner.GoogleSub == "" {
			d.Owner.ID = uuid.NewString()
			d.Owner.GoogleSub = i.Subject
		}
		if d.Owner.GoogleSub != i.Subject {
			return errDenied
		}
		id = d.Owner.ID
		return nil
	})
	return id, err
}
func (s *Store) newPending(authID string) (*Pending, error) {
	p := &Pending{ID: randomToken(), CSRF: randomToken(), AuthID: authID, Expires: time.Now().Add(5 * time.Minute)}
	err := s.update(func(d *diskState) error {
		if d.Owner.ID == "" {
			return errDenied
		}
		if len(d.Pending) >= 100 {
			return errDenied
		}
		d.Pending[p.ID] = p
		return nil
	})
	return p, err
}
func (s *Store) pending(id string) (*Pending, error) {
	var out Pending
	err := s.update(func(d *diskState) error {
		p := d.Pending[id]
		if p == nil {
			return errDenied
		}
		out = *p
		return nil
	})
	return &out, err
}
func (s *Store) enrollment(id string) (string, error) {
	var secret string
	err := s.update(func(d *diskState) error {
		p := d.Pending[id]
		if p == nil {
			return errDenied
		}
		if d.Owner.Seed != "" {
			return nil
		}
		if p.Seed == "" {
			key, e := totp.Generate(totp.GenerateOpts{Issuer: "Jomon", AccountName: s.email})
			if e != nil {
				return e
			}
			p.Seed = s.seal(key.Secret())
		}
		var e error
		secret, e = s.unseal(p.Seed)
		return e
	})
	return secret, err
}
func validSession(v *Session, now time.Time) bool {
	return v != nil && now.Before(v.Expires) && now.Sub(v.LastSeen) < time.Hour
}
func (s *Store) session(id string) (*Session, error) {
	var out Session
	err := s.update(func(d *diskState) error {
		v := d.Sessions[id]
		if !validSession(v, time.Now()) {
			return errDenied
		}
		v.LastSeen = time.Now()
		out = *v
		return nil
	})
	return &out, err
}

func (s *Store) finishMFA(id, input string) (*Session, []string, error) {
	var result *Session
	var codes []string
	denied := false
	err := s.update(func(d *diskState) error {
		p := d.Pending[id]
		now := time.Now()
		if p == nil || now.Before(d.Owner.LockedUntil) {
			return errDenied
		}
		seed := d.Owner.Seed
		enrolling := seed == ""
		if enrolling {
			seed = p.Seed
		}
		ok := false
		step := now.Unix() / 30
		if seed != "" && len(input) == 6 {
			secret, e := s.unseal(seed)
			if e != nil {
				return e
			}
			for delta := int64(-1); delta <= 1; delta++ {
				candidate := step + delta
				if candidate <= d.Owner.LastStep {
					continue
				}
				code, e := totp.GenerateCode(secret, time.Unix(candidate*30, 0))
				if e == nil && constantEqual(code, input) {
					ok = true
					d.Owner.LastStep = candidate
					break
				}
			}
		}
		if !enrolling && d.Recovery[hash(input)] {
			delete(d.Recovery, hash(input))
			ok = true
		}
		if !ok {
			d.Owner.Failures++
			if d.Owner.Failures >= 5 {
				d.Owner.LockedUntil = now.Add(5 * time.Minute)
				d.Owner.Failures = 0
			}
			denied = true
			return nil
		}
		if enrolling {
			d.Owner.Seed = seed
			for range 10 {
				code := randomToken()
				codes = append(codes, code)
				d.Recovery[hash(code)] = true
			}
		}
		d.Owner.Failures = 0
		d.Owner.LockedUntil = time.Time{}
		result = &Session{ID: randomToken(), CSRF: randomToken(), AuthTime: now, LastSeen: now, Expires: now.Add(12 * time.Hour)}
		d.Sessions[result.ID] = result
		delete(d.Pending, id)
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	if denied {
		return nil, nil, errDenied
	}
	return result, codes, nil
}
func (s *Store) logout(id string) error {
	return s.update(func(d *diskState) error {
		delete(d.Sessions, id)
		for k, v := range d.Tokens {
			if v.SessionID == id {
				delete(d.Tokens, k)
			}
		}
		for k, v := range d.Auth {
			if v.SessionID == id {
				delete(d.Auth, k)
			}
		}
		return nil
	})
}
func (s *Store) newHandshake(authID string) (*Handshake, error) {
	h := &Handshake{ID: randomToken(), State: randomToken(), Nonce: randomToken(), Verifier: randomToken(), AuthID: authID, Expires: time.Now().Add(5 * time.Minute)}
	err := s.update(func(d *diskState) error {
		if len(d.Handshakes) >= 200 {
			return fmt.Errorf("too many pending logins")
		}
		d.Handshakes[h.ID] = h
		return nil
	})
	return h, err
}
func (s *Store) consumeHandshake(id, state string) (*Handshake, error) {
	var out Handshake
	err := s.update(func(d *diskState) error {
		h := d.Handshakes[id]
		if h == nil || !constantEqual(h.State, state) {
			return errDenied
		}
		out = *h
		delete(d.Handshakes, id)
		return nil
	})
	return &out, err
}
