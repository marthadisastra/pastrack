# Trip History

Trip History, vehicle trip tabs and playback links now use a dedicated history page. The prior generic template incorrectly rendered individual, paginated GPS positions as trip rows, fetched only 50 fixes and could mix vehicles when no filter was chosen.

Authenticated `GET /api/v1/trips/history?vehicle_id=...&from=...&to=...` requires one vehicle and a range of at most 31 days. It reads up to 20,000 validated GPS fixes and explicitly rejects overflow. The reporting device with most fixes is selected; duplicate GPS times are removed. It returns full chronological positions, derived journey segments, the selected vehicle and its latest validated GPS time. It does not create or modify telemetry or persisted trips.

Journeys use movement ≥3 km/h, stopping for ≥3 minutes, a minimum observed distance of 100 m and a maximum inferred/reported speed of 160 km/h. Gaps over five minutes and invalid samples split journeys. An end at the requested boundary is explicitly labeled an excerpt, not a completed trip. Distances and durations reflect observed data only.

The page provides vehicle and inclusive WIB date filters, full-route map, journey list, Play/Pause, restart, playback speed and time slider. A journey's Putar action restricts the player to its time span. Empty ranges offer the latest GPS date. The October 31 records for B 9101 PTR and B 9102 PTR use the same vehicle/date filters and ordinary database relations as other history. There is no alternate data-source selector. Imported records retain internal origin metadata and are documented in fuel-events-workspace.md.

Validation includes MySQL vehicle/device isolation and complete route retrieval, segmentation/gap/boundary unit tests, 41 DOM tests, and browser checks of actual latest-date selection, simulated journeys, Play/Pause/seek and desktop/mobile layouts. Deployment uses `deploy/rebuild/deploy_trip_history.py`, backs up the current binary, replaces only the managed web application and checks both web and receiver health. No migration or receiver restart is required.
