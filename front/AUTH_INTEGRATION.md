# Go authentication — stage 1

The live Vue application is served at `/app/`; the existing Go interface at `/` remains available. Existing `users`, bcrypt passwords, roles and SQL sessions are reused. Username or email can be used to sign in. Duplicate email matches are rejected and require administrator resolution. Recovery requires a valid, unique email on the existing account.

Authentication defaults to Go (`VITE_AUTH_MODE=server`) independently of `VITE_USE_MOCK=true`. Fleet values, module management, branding, audio and RBAC screens still use their existing demo adapters in this stage. SMTP configuration uses Go and SQL. Use `VITE_AUTH_MODE=demo` only for isolated UI development. Never put SMTP passwords or server encryption keys in Vite variables.

## Settings → SMTP & Email

Sign in as an existing Go administrator. Configure host, port, STARTTLS (usually 587) or implicit TLS (usually 465), username, password and sender. Enable SMTP and save. Test Connection verifies transport and authentication without sending an email. Saving configuration alone does not verify delivery or sender authorization; test a recovery request using an account email you control after setup.

SMTP is initially disabled. Forgot Password reports that recovery is unavailable until SMTP is configured. Credentials are AES-256-GCM encrypted in `auth_smtp_settings`; the encryption key lives only in the server environment. GET never returns the password. Leaving its field blank preserves the current password; Clear Password removes it. Configuration is admin-only and is not stored in browser localStorage.

## API and sessions

- `POST /api/v1/auth/login` `{email|username,password,remember}`
- `GET /api/v1/auth/me`
- `POST /api/v1/auth/logout` `{}`
- `POST /api/v1/auth/forgot-password` `{email}`
- `POST /api/v1/auth/reset-password` `{token,password}`
- `GET/PUT /api/v1/auth/smtp`
- `POST /api/v1/auth/smtp/test`

Cookies are HttpOnly, SameSite=Strict and Secure on live HTTPS. Default session lifetime is 12 hours with a browser session cookie; Remember Device persists up to 30 days. SQL stores only the token hash. Refresh restores the session from `/me`; protected routes redirect to login, and active pages revalidate on focus/visibility and every minute. Logout invalidates the current SQL session. No authentication token is stored in localStorage.

Reset links expire after 15 minutes and are single-use. Link tokens appear in the URL fragment, so they are not sent in HTTP request URLs. Only token hashes are persisted for validation. The mail queue temporarily encrypts delivery links, retries up to five times and clears encrypted payloads on success, expiry or final failure. Changing password increments `auth_version` and revokes every existing session and reset link. Passwords require 12–72 UTF-8 bytes including upper/lowercase letters, a digit and a symbol. No automatic login after reset.

Mutation requests require JSON and reject cross-origin requests. Login, reset and recovery requests are rate limited in process memory. Restart resets counters; multiple replicas require a shared limiter in a later deployment. Recovery returns a generic response for unknown/duplicate/inactive accounts when SMTP is ready. SMTP sending is handled asynchronously by Go.

Local `npm run dev` proxies auth to the live Go service over HTTPS by default, so existing accounts work in the local template too. Loopback development translates same-origin requests to the upstream origin and removes Secure only on the loopback HTTP response cookie; live cookies retain Secure. Set `AUTH_PROXY_TARGET=http://127.0.0.1:18094` for an isolated local Go instance. Local login and SMTP settings therefore affect the existing live authentication database by default.

## Server configuration and deployment

`AUTH_APP_URL=https://toko.azlaundry.my.id/app`, `APP_FRONTEND_DIR=/www/wwwroot/pastrack/front`, `AUTH_MAIL_KEY` (base64 of 32 random bytes), `COOKIE_SECURE=true`. Preserve the encryption key in protected backups; replacing it prevents decrypting saved SMTP credentials and pending mail. Do not expose or commit it. SMTP can optionally come from server environment variables, but the Settings database row takes precedence.

Migration `017_auth_password_recovery.sql` is additive. `deploy/rebuild/deploy_auth_settings.py` stages the web binary/frontend, applies only this migration, grants the two new tables, backs up the web binary/environment and rolls back the web deployment on failed health checks. It does not restart the GPS receiver or alter existing account passwords.

## Verification

`go test ./...`

Local database integration is restricted to `127.0.0.1:33307/pastrack_test_local`. Set `PASTRACK_TEST_DSN` with `parseTime=true&time_zone=%27%2B00%3A00%27` and run `go test ./internal/platform -run 'TestAuthRecovery|TestSMTP' -count=1`. Add `RUN_AUTH_BROWSER=1` to run the real Vue/Go browser flow; this starts a temporary Vite server on 5186 and uses an in-process test mail transport. No external email is sent. Existing frontend regression tests run in explicit demo authentication mode on port 5175.

Design references: [OWASP Forgot Password](https://cheatsheetseries.owasp.org/cheatsheets/Forgot_Password_Cheat_Sheet.html) and [OWASP Session Management](https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html).
