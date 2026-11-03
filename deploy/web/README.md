# Private web deployment

The web module is independent of Wails. Build from `web/` using its Dockerfile,
adding `--label org.opencontainers.image.revision=<verified-commit>`. Retain the
image ID and source manifest in deployment metadata. Test the native platform
and concurrency behavior before switching application authentication.

Use `compose.yml` in a private runtime directory. Copy the shape from
`web/config.example.json`, then provision real values outside source control.
The runtime state directory is mode0700 and owned by UID65532; config and secret
files are mode0400 owned by that UID. Keep backups of state, master key, signing
key, client secrets and owner configuration together under restricted access.
Use SQLite's online backup API or stop the portal briefly for a consistent copy.
Do not generate replacement keys or reset the owner pin on restart.

The container uses host networking with a mandatory loopback listener at
127.0.0.1:8087; it exposes no public application port. nginx terminates TLS and
checks the host independently. The supplied nginx configuration preserves the
app's CSP and Referrer-Policy; overriding them with `no-referrer` or a narrower
`form-action` can break native MFA/logout form submissions.

Stage DNS and the HTTP ACME challenge location first. Obtain a certificate for
nav.jomon.dev with the existing ACME account, retain automatic renewal, then
enable the TLS site. Validate nginx before reload. Set `NAV_IMAGE` to the tested
release and start the portal with `docker compose --env-file compose.env up -d`.
Check `/healthz` with Host nav.jomon.dev through both loopback and public TLS.

Configure the Google web client with exactly
`https://nav.jomon.dev/auth/google/callback`. Only `openid email profile` is
needed. Configure the sole permitted owner privately; the first verified Google
subject becomes permanently pinned. The owner completes the authenticator
enrollment in the browser and stores the one-use recovery codes.

PanSou and Quota require separate client secrets, their native SSO mode, and the
exact callbacks in `web/config.example.json`. Preserve existing local password,
MFA and application master-key files. Keep current databases; do not restore an
older OAuth-credential snapshot as an automatic rollback. Before cutover, verify
the Google+MFA chain, callbacks, direct-access denial and global logout. Keep a
local-auth rollback configuration until acceptance has completed.

Acceptance must also check that arbitrary identity headers, old local cookies,
invalid state/nonce, revoked sessions and a provider outage cannot authorize app
requests. Do not trigger application business actions just to test authentication.
