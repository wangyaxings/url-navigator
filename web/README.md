# Jomon 导航 Web

Independent Go web module; the Wails desktop edition is unchanged. The issuer is fixed to `https://nav.jomon.dev`. Google authenticates the privately configured owner, followed by mandatory local TOTP or a single-use recovery code. The first verified Google subject is pinned permanently to a stable internal UUID.

## Configuration and running

Copy `config.example.json` to a private location, then set the Google web OAuth client ID and secret-file paths. Register only `https://nav.jomon.dev/auth/google/callback` in Google. The owner email file contains the exact allowed email (case-sensitive). Client secret files contain distinct random values of at least 32 characters; use the same respective secret in PanSou and Quota. The master-key file contains exactly 32 **raw random bytes**. The signing-key file contains a persistent RSA private key (PKCS#1 or PKCS#8 PEM, at least 2048 bits). Read all secrets from read-only mounts; never put actual secrets or the owner email in this repository.

Example key preparation on the deployment host, inside an already private directory:

```sh
umask 077
openssl rand -out nav_master_key 32
openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:3072 -out nav_signing_key.pem
openssl rand -hex 32 > nav_pansou_secret
openssl rand -hex 32 > nav_quota_secret
```

Build and run from `web/`:

```sh
go test ./...
go vet ./...
go test -race ./...
go build -trimpath -o nav .
./nav -config /etc/nav/config.json
docker build -t jomon-nav:local .
```

The container runs as UID/GID **65532:65532**. Create the state directory mode 0700 owned by that UID and allow it to read secret/config files. Use **host networking on Linux** so the mandatory loopback listener remains reachable by host nginx; do not publish a container port. Example runtime shape:

```sh
docker run -d --name jomon-nav --restart unless-stopped --network host \
  --read-only --cap-drop ALL --security-opt no-new-privileges \
  -v /srv/nav/state:/var/lib/nav \
  -v /srv/nav/secrets:/run/secrets:ro \
  -v /srv/nav/config.json:/etc/nav/config.json:ro jomon-nav:local
```

Keep a single portal instance per SQLite database. The single-owner state is a SQLite transactional aggregate, pruned on access; authorization/state records are bounded to prevent unbounded unauthenticated growth. No state, secret, signing key, or owner pin is regenerated on startup. Changing the master/signing key is an explicit operational migration; retain both with backups. Startup checks that the current master key can decrypt existing MFA state.

## Reverse proxy and protocol

nginx must terminate TLS, forward `Host: nav.jomon.dev` to `127.0.0.1:8087`, strip visitor identity headers, and reject other server names/default direct-IP requests. The portal independently rejects other Host values and never accepts visitor identity headers. Add HSTS at nginx after verifying HTTPS. Use access logs without query strings on authentication routes (OAuth codes occur in callback query parameters). Apply ordinary request/IP rate limits at nginx; the portal independently rate-limits MFA failures per owner (five failures lock authentication for five minutes).

`GET /healthz` requires the same Host header. Discovery at `/.well-known/openid-configuration` advertises `/authorize`, `/oauth/token`, `/oauth/introspect`, `/keys`, and `/logout`. Downstream clients must use discovery, client_secret_basic, code + PKCE S256, state and nonce. The only registered clients/callbacks are `pansou` / `https://pan.jomon.dev/auth/sso/callback` and `quota` / `https://quota.jomon.dev/auth/sso/callback`. Only `openid email profile` scopes and the authorization-code grant are allowed. No refresh tokens or JWT/client-credentials grant.

RS256 ID tokens include `sub`, verified `email`, `sid`, `auth_time`, `nonce`, `aud`, and `acr=urn:jomon:acr:google-totp`. The public `sid` is derived from the private session credential and cannot be replayed as the portal cookie. Introspection requires the registered client's Basic credentials and only reveals that client's own token; its active response includes `sub`, `client_id`, `aud` (array), `exp`, `sid`, `acr`, `auth_time`. Each app must introspect every protected request without a positive cache. Browser sessions expire absolutely after 12 hours and after one hour idle. Introspection updates idle activity but never the absolute deadline. ID tokens last five minutes; requests within six minutes of absolute expiry require fresh Google + MFA login. Access tokens expire with their parent and become inactive immediately on logout.

Application logout should redirect to `https://nav.jomon.dev/logout`; GET only displays confirmation. The page posts a session-bound CSRF token with a matching Origin to revoke the parent and linked tokens atomically. No GET route revokes a session.

## Verification and preservation

Tests use isolated SQLite files and test-only fake upstream identity; production has no bypass flag or alternate upstream issuer. Tests exercise real provider endpoints, PKCE/code redemption, signed ID-token claims, introspection and logout; real Google JWT verification is tested against a local test JWKS for invalid signature/issuer/audience/expiry/nonce. Concurrency tests cover recovery codes and authorization codes. Real Google consent, physical authenticator enrollment, nginx TLS, and application cutover still need deployment verification with the owner's credentials.

Before changes or rollback, stop this single-instance portal briefly and copy the SQLite database **and any WAL/SHM files** together, or use SQLite's online backup API. Back up the external master key, signing key, client secrets and owner configuration separately with restricted access. Preserve existing application databases and legacy MFA/secrets; this module never opens them. Returning an app to local authentication is an app configuration change, not a database reset.

Protocol references: [Zitadel OIDC storage contract](https://github.com/zitadel/oidc/blob/main/pkg/op/storage.go), [Google OpenID Connect](https://developers.google.com/identity/openid-connect/openid-connect).
