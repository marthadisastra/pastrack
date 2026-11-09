# Ignition indicators

Visible ON/OFF badges use ignition_status decoded from last device report: GTIGN ON, GTIGF OFF, GTIGL/GTVGL report type plus supported virtual events. The backend already selects original device send time, preserves last state until the next report and separates GPS position time. No speed-based inference or expiry. Physical wiring metadata is preserved independently.

Updated dashboard/vehicle table indicators, selected-vehicle identity and live map popups. ON green, OFF red, missing report explicit. SSE updates badges per event. Added regression for supplied GTIGF with stale no-fix GPS and report time 10:52:24 UTC, plus UI OFF-to-ON SSE test. 36 UI tests pass; platform/protocol race tests and platform vet pass. No physical output command sent.
