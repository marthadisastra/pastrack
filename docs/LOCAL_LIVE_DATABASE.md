# Local application linked to the live database

The user explicitly requested identical local/live data and permissions. The earlier proposed read-only restriction is disabled.

- Local application: http://127.0.0.1:18090
- Live application: https://toko.azlaundry.my.id/
- Shared database: pastrack_v2_20261103_r2, server MariaDB port 3306.
- Connection: encrypted SSH forwarding from 127.0.0.1:33308 to server 127.0.0.1:3306; no public MySQL port was opened.
- Local private environment: .local/live-view.env, mode 0600. No production password is embedded in source.
- Login uses the same existing live account/password. Application roles remain the same; this is not viewer-only mode.

Normal data reads and session authentication use the live SQL connection. Business mutations, report storage/download and SSE are routed to the authoritative live application. This keeps business data, generated files and notifications consistent across both URLs instead of creating laptop-only files or isolated event streams. Local assets are built from the same source deployed to the server.

Saving a business change locally changes live production data. The dedicated tunnel DB account has the same table privileges as the live runtime; immutable raw messages retain SELECT/INSERT only. No migrations, development seeds, account bootstrap or second GPS receiver are run in local-live mode.

Start: `./scripts/local-live-start.sh` from the repository root. It opens the SSH tunnel using the existing authorized key, builds the local web app and waits for health. Use `./scripts/local-stop.sh` to stop local web; the foreground supervisor then closes its tunnel. Keep the start command running. Its log files are private under .local/.

The separate isolated MySQL on 33307 and test schema remain available for tests. `scripts/local-start.sh` is still the isolated-development startup; use `scripts/local-live-start.sh` for the shared live data requested here.

Verified: live database/schema identity and data counts through the tunnel; local identity read_only=false; live vehicle reads; shared report GET; shared SSE ready event; an intentionally empty registration POST returns the same validation error as live without writing business data. Temporary smoke-test session was logged out afterward.
