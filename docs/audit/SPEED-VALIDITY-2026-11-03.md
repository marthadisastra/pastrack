# Speed validity correction

Observed GPS positions 40194–40208 carried raw speed 13.0 km/h, no_fix, repeated GPS time 2026-11-03 10:32:29 UTC and unchanged coordinates despite new server receipts until 10:36:31 UTC. Current-speed views incorrectly displayed that retained speed without a validity warning.

Dashboard/vehicle tables, selected vehicle speed card, tracking list and map labels now suppress current numeric speed when GNSS is invalid or explicitly stale. Stored raw speed and history remain unchanged. No zero speed inferred.

35 UI tests passed, including invalid GNSS with raw speed 13. Live web deployed and local 18090 restarted; served assets verified. No physical remote command transmitted during checks.
