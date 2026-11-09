# Pastrack Go receiver

TCP receiver for Queclink GV30CEU. Replaces the ingestion path in `pastrack.py`, without Beego. Uses the existing MySQL/MariaDB schema.

## Run

Go 1.24 or later:

```sh
go test -race ./...
go build -o pastrack .
cp .env.example .env
# Set database credentials in .env, then:
set -a
. ./.env
set +a
./pastrack
```

The application reads environment variables. It does not automatically load `.env`; the aaPanel launch script does.

- Tracker TCP: `0.0.0.0:5001`
- Local health endpoint: `http://127.0.0.1:18087/healthz`
- Durable queue: `/opt/pastrack/spool` (must be writable by runtime user)
- DB: `gps_tracker_rnd`, existing `gps_devices`, `device_raw_messages`, `gps_positions`, `device_events`. `gps_positions.device_send_time` must exist.

## Protocol coverage

Frames end in `$`; split TCP reads and multiple frames per read are supported. Connections have a 300 second idle deadline, a 60,000 byte frame limit, and a 256 connection limit.

Position extraction is limited to the observed 20-field GV30CEU `GTSTT` layout with protocol version `80201E0100`. `+RESP:` and `+BUFF:` are accepted. Coordinates, finite numeric values, heading and timestamps are checked. Device times are interpreted using `GPS_TIMEZONE` (UTC by default), then stored in UTC. Accuracy and satellite fields remain unverified and are not guessed. Other report layouts retain raw payload and fields with a `partial` status. Malformed reports are stored as `failed`. Unknown/inactive IMEIs do not create devices or positions automatically.

CFS300B is a BLE/iBeacon capacitive fuel sensor. The receiver accepts and retains Queclink reports forwarded by the tracker; **fuel level, liters, tilt, sensor identity and fuel theft decoding are not implemented yet**. Need the GV30CEU BLE report protocol/example payload and tank calibration to implement them safely. Reference: https://www.queclink.com/wp-content/uploads/2025/10/ENCFS300B-datasheet-250818.pdf

No server ACK or tracker configuration command is sent: required behavior must be confirmed against the device's protocol manual and server-ACK configuration.

## Persistence and recovery

Each complete frame is written and fsynced to a private spool file before DB processing. A worker retries queued messages if the DB is unavailable. Raw messages, positions and events are committed together. Delivery is **at least once**: a crash after DB commit but before deleting the spool file can create a duplicate. Monitor disk capacity and `queued` in health output. DB errors leave frames queued and are logged; an unrecoverable schema/data error can block the queue until fixed. Incomplete or oversized frames are rejected. This receiver handles ASCII Queclink frames, not binary tracker protocols.

`/healthz` returns 503 if MySQL is unavailable and reports queue length otherwise. The endpoint is local only. Health success does not certify that the fuel sensor is paired or reporting.

## Deployment / aaPanel

Production root: `/opt/pastrack`, aaPanel Go Project: `pastrack`. Use user `www`, executable `/opt/pastrack/pastrack`, command `/opt/pastrack/deploy/run-aapanel.sh`, TCP port `5001`, boot enabled, HTTP proxy disabled. TCP tracker traffic must reach port 5001 directly. Credentials remain on server in `.env` with mode 0600.

Build Linux binary:

```sh
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o pastrack-linux-amd64 .
```

Preserve `.env` and `spool/` on future deployments. Stop the project through aaPanel, atomically replace the binary, then start and check health, aaPanel runtime state and newly received device records. Do not run the Python and Go receivers on the same port.

The legacy Python source remains unchanged locally and at `/opt/queclink/pastrack.py` on the server as a rollback reference. To roll back, stop the Go project first and run the legacy Python receiver with the preserved DB environment from `/opt/pastrack/.env`.

## Verified production installation

Server `76.13.195.48`, aaPanel Go Project `pastrack` (ID 69), runtime user `www`, start-on-boot enabled. Public TCP port 5001 was checked from the deployment laptop. Health remains local on port 18087. Database credentials were reused on-server from the old receiver without copying them to the local project.

`deploy/register-aapanel.py --legacy-pid PID` is a one-time cutover helper, executed with aaPanel's Python from `/www/server/panel`; it refuses an existing project. Do not re-run it for normal updates. `deploy/verify-staging.py` is the pre-cutover integration check; it requires the staging listener and MySQL connector environment used during installation.

## Web dashboard (Beego)

The separate `web/` Go module implements the Beego monitoring UI. Production: https://toko.azlaundry.my.id/ , deployed to `/www/wwwroot/pastrack`, aaPanel Go Project `pastrack_web` (ID 70). See `web/README.md` for login configuration, deployment and feature limits. Web deployment does not restart the tracker receiver.

## Event-driven web updates

The receiver's spool worker is woken immediately after a durable enqueue; it no longer waits for a one-second processing tick. After successful persistence/removal, it signals a dedicated publisher, which sends a `positions` invalidation over the private Unix socket configured by REALTIME_SOCKET_PATH (production `/opt/pastrack/runtime/live.sock`). Only failed persistence/delivery is retried with a timer; idle processing does not poll. Startup emits a recovery invalidation. The web service relays these commits over authenticated SSE without polling the GPS tables. Deploy both services, preserve spool and private environment files, and keep the socket directory owned by runtime user www with mode 0700. See web/README.md for the full update contract.
