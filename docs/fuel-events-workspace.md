# Fuel Events workspace

Fuel Events is a compact, synchronized route and fuel workspace in the v2 application. The only filter fields are Kendaraan (All and the vehicle list), Start date and End date. Dates include the entire selected start/end days in WIB. GPS and fuel readings are resolved through vehicle relations, without fleet, sensor or tank controls. The vehicle picker fits its content; only fuel volume is plotted. Speed remains a numeric playback readout, not a second chart series.

On a 1366×768 desktop, the map and chart sit side by side with the shared player visible below. Summary cards are compact; event details and detection rules are expandable. The map observes container resizes. Play/Pause, restart, ±1 minute, speed and scrubber use the same clock for position, traveled route, graph cursor/reveal and fuel level. Source gaps, invalid readings and calibration changes are not bridged. Mobile reflows the same controls vertically.

`GET /api/v1/fuel/events/workspace` reads stored events, readings, validated GPS, sensors, geofences, enabled rules and consumption summaries. Limits are explicit: 1,000 events/geofences and 20,000 readings/positions. All returns events across vehicles without combining GPS tracks or fuel readings. A selected vehicle returns its associated history and enables playback. Focusing an event from All selects that event’s vehicle before seeking playback. Consumption summaries are added only if their periods do not overlap. Event selection seeks the player; selecting a vehicle reads its normal stored data. There is no alternate data-source selector or generated-data endpoint.

`POST /api/v1/fuel/events/:id/investigation` remains available to admin/operator with optimistic row versions and transactional audit. Viewers can inspect and download the escaped HTML BAP snapshot. Manual receipt metadata is not an external SPBU verification. Physical remote output is not exposed here.

## October 31 records

At the user's explicit request, `deploy/rebuild/import_october_journeys.py` imports two complete sets of normal database relations: vehicles B 9101 PTR and B 9102 PTR, GPS devices/assignments, tank profiles, sensors/bindings/calibrations, raw messages, parsed positions/telemetry/readings, latest-state rows, persisted trips, consumption summaries and a fuel-drop event for B 9102 PTR. Internal provenance and audit retain origin; there is no separate display mode. Existing vehicles and their telemetry are not modified.

Each vehicle has 721 samples, every 30 seconds from 31 October 2026 06:00 to 12:00 WIB. Coordinates follow a road route Jakarta–Cirebon obtained from OSRM. Both trips start with 320 L in a 400 L tank, consume fuel proportionally to distance and include stops. B 9102 PTR additionally loses 40 L from 09:00 to 09:02. The import is atomic, refuses collisions with unrelated plates, validates completed imports before skipping, and is run only after a private database backup. `deploy/rebuild/deploy_fuel_focus.py` verifies ordinary API reads for both vehicles and the removed endpoint, atomically replaces only the managed web binary, and checks receiver health.

Validation: Go tests, isolated MySQL API tests, a fresh database import and repeat-import check, 42 DOM tests, desktop viewport bounds checks, one fuel-series assertion, ordinary vehicle/event reads, playback and mobile overflow checks. Browser evidence is under `output/fuel-events`.

Product scope: additional features or controls outside the agreed vehicle/date flow require user confirmation before implementation.
