# Navigation portal and application SSO

## Approved outcome

Add a web edition to URL Navigator at `https://nav.jomon.dev`, preserving the existing Wails desktop edition. Only the configured Google owner may log in, followed by portal TOTP or a recovery code. PanSou and Quota use that login without their own password/MFA prompts. Direct application access remains protected. Preserve all existing application data, keys, and MFA state for rollback.

## Architecture and protocol contract

The portal is a small standalone Go module under `web/`, with a maintained OIDC provider library (`github.com/zitadel/oidc/v3/pkg/op`), a SQLite database, embedded web assets, Google OIDC as upstream identity, and portal MFA. Root desktop sources/build remain independent. Initial navigation cards are configured by deployment; the web edition need not duplicate desktop bookmark editing in this change.

- Fixed issuer: `https://nav.jomon.dev`; discovery at `/.well-known/openid-configuration`.
- Upstream Google callback: `https://nav.jomon.dev/auth/google/callback`.
- Downstream confidential client IDs: `pansou`, `quota`, each with a distinct secret.
- Exact registered callbacks: `https://pan.jomon.dev/auth/sso/callback` and `https://quota.jomon.dev/auth/sso/callback`.
- Authorization code flow with PKCE S256, state, nonce; scopes `openid email profile`; no refresh tokens initially.
- Stable internal owner UUID is `sub`. Claims: verified `email`, `sid`, `auth_time`, and `acr=urn:jomon:acr:google-totp`.
- ID tokens signed by persistent provider key; clients validate issuer, signature, audience, expiry, nonce, subject and ACR. Access tokens remain server-side.
- RFC 7662 introspection requires client authentication and returns `active`, `sub`, `client_id`, `aud`, `exp`, `sid`, `acr`. Token is active only while its parent session is valid. Each protected app request checks introspection without a positive cache; network failures return 503, invalid sessions return 401/login.
- Sessions: absolute 12 hours, idle 1 hour. Introspection touches parent activity, but never extends absolute expiry. Tokens cannot exceed parent expiry.
- Host-only opaque cookies: `__Host-nav_session`, `__Host-pan_session`, `__Host-quota_session`; Secure, HttpOnly, Path=/, SameSite=Lax, no Domain. Pending handshake cookies are also host-only, short-lived and Secure.
- Central logout: CSRF-protected POST revokes parent session and all linked tokens atomically. App logout deletes local session and redirects browser to a portal logout confirmation page; that page performs CSRF-protected POST. No state-changing GET logout. Cookies remaining on another host no longer authorize.
- App auth mode `local|sso`, default local. SSO mode disables local password/MFA endpoints, rejects old local cookies, and never initializes or rewrites legacy MFA state. Existing secrets/config are retained. All public pages, APIs, downloads and legacy routes are covered except deliberate health/static/OIDC callback endpoints.
- Unauthenticated navigational GET redirects to SSO; APIs return JSON 401; failed introspection returns 503. Do not redirect/replay mutations. Validate return paths as local paths, rejecting scheme-relative URLs and backslashes/control chars.
- Preserve app CSRF protections. Require configured public origin plus session-bound CSRF for cookie-authenticated mutations. Do not trust visitor identity headers; nginx strips them. No domain-wide cookies or unsigned proxy identity.

## Google and MFA requirements

Validate Google's JWT signature, issuer, audience, expiration and nonce. Require `email_verified=true` and exact private-config allowlisted email. Atomically pin the first verified Google `sub` to the single owner; reject account substitution afterward. Never commit the owner's email or OAuth credentials in this public repository.

Google callback creates only a short pending MFA session. No portal/application data or authorization code before successful MFA. Initial setup requires a new TOTP challenge; show recovery codes once. Encrypt TOTP seeds at rest with an external master key, hash recovery codes, consume once, reject reused TOTP time steps, rate-limit failures. Use generic failures, no secret/token logging, no secrets in query strings except standard short-lived OAuth codes. OAuth state and codes are one-use under concurrent requests.

## UI

Simple Chinese interface: Google login, TOTP enrollment/challenge, recovery-code alternative, navigation cards and logout. Avoid implementation details and diagnostic clutter. Initial cards PanSou and Quota; configure any future cards on the server. Show no application links to unauthenticated/pending-MFA users. Responsive and accessible, with visible errors and pending states. Preserve the desktop edition.

## Deployment and preservation

Run portal on the existing AWS ARM server behind nginx TLS, loopback-only listener, with persistent state and read-only secret files. Add only `nav.jomon.dev` DNS. Stage and verify central login before changing apps to SSO. Back up databases consistently and retain existing application secret files, MFA data, and local-mode rollback configs. Never reset databases or modify subscription/transfer records. No Baidu manual checks or transfers are required for this authentication change. Audit old direct-IP entry points and prevent them bypassing the chosen authentication.

## Acceptance

Allowed Google owner plus TOTP reaches portal and both applications; a different Google account, missing MFA, forged headers, old app cookies, direct-IP requests, invalid state/nonce, expired/revoked sessions and CSRF attempts do not. Logout immediately invalidates both applications. Provider outage fails closed. Existing app business tests and desktop build still pass. Run targeted concurrency tests for code/recovery-code consumption and logout versus introspection. Verify browser layout and actual login with user-owned credentials only. Record commit/image digests, backups and before/after preserved-data evidence.
