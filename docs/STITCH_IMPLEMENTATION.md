# Stitch HTML implementation — 3 November 2026

The local v2 application now serves the imported HTML compositions from `Desain/`, replacing the previous generic table-based application shell.

## Implemented

- `scripts/design/import-stitch.cjs` imports all 21 exported page bodies, their class names, table headers and exact 17-entry sidebar order. `stitch-pages.json` records the source file for each composition.
- `scripts/design/tailwind.config.cjs` extracts the original shared theme. Tailwind CSS is compiled at build time; no runtime CDN or inline script is required.
- Inter and Material Symbols fonts are served from local font files with their licenses. `stitch-export.css` preserves exported styles; `stitch.css` adds responsive navigation, live-data states and dialogs.
- `stitch.js` binds real APIs to the exported table/card/map/chart positions. Vehicle overview, trips, fuel, calibration and device tabs use their own exported layouts, rather than a shared replacement screen.
- Original demo table rows, sample coordinates/paths, plots, identity examples and numeric telemetry are removed from imported templates. Current data is supplied by API; missing values remain N/A. Physical ignition remains NOT_INSTALLED.
- Existing 17 sidebar entries retained; development resources no longer appear as extra top-level menus. Inventory/configuration actions remain accessible through contextual forms/dialogs.
- Live map, filter/search, keyset pagination, unit selection, unit registration/editing, user management, alert acknowledgement, raw inspection, measured calibration points/approval and report creation/download use real handlers.
- GPS playback controls use validated recorded points, preserve gaps and expose speed/seek. Circle/polygon drawing opens the real geofence editor. Actual stored geofence geometry is drawn on the map.
- Report backend exports CSV, XLSX and PDF with tests for blank fuel values and valid artifact format. GPS GPX/KML exports contain only the currently loaded recorded points.

## Visual and functional boundaries

This implementation preserves the exported DOM/layout as its source. It does **not** claim pixel-perfect acceptance of all screens or that every mockup feature is supported. Actual data and empty states necessarily differ from the filled demo screenshot.

Hardware commands (immobilizer, OTA, reboot/poll), unconfigured external communications and unsupported sensor processing remain disabled with explicit reasons. Enabling a mockup button is not proof of a supported command. Ignition or raw-frequency units are never inferred to match labels in a mockup.

The overview was visually inspected at mobile and desktop sizes before browser access was disabled in Work settings. Subsequent visual verification of every page could not be completed. DOM tests cover all 17 primary navigation targets and all five vehicle detail compositions. They are not a replacement for complete visual acceptance.

## Build/test

```sh
npm ci --prefix scripts/design
npm run build --prefix scripts/design
node --test tests/platform.test.cjs
PASTRACK_TEST_DSN='root@tcp(127.0.0.1:33307)/pastrack_test_local?parseTime=true' go test -race ./...
```

Restart the local application after assets are rebuilt because assets are embedded in the binary. No production deployment or destructive migration is part of this change.
