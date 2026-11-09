# Pastrack — Queclink device management assessment

Assessment: 29 October 2026. Device: GV30CEU, IMEI ending 93813. Earlier Manage Tool screenshots: version V1.5.91, BB GV30CEUR01A01V22M128, protocol report 80201E0100. These observed values must be checked against the applicable protocol revision before commands are enabled.

## Verified sources and limits

- Queclink development/integration: https://www.queclink.com/development-integration — officially offers QMS for remote device update/management, QPAK and @Track Protocol Pro. This does not prove that this particular GV30CEU firmware supports Protocol Pro; its received frames currently use ASCII @Track framing.
- Queclink QMS: https://www.queclink.com/development-integration-detail/QMS — indexed official descriptions list device/task monitoring, access controls/audit, firmware management, bulk configuration and customer-hosted operation. The current site links to lowercase /qms, whose content could not be fetched during this review.
- Manufacturer GV30CEU datasheet, indexed at https://www.queclink.com/wp-content/uploads/2026/01/EN-GV30CEU-datasheet-250818.pdf — TCP/UDP/SMS, scheduled reporting, BLE gateway, and OTA digital-output control. The legacy PDF URL now returns 404, so the indexed datasheet is not a substitute for a current model-specific command manual.
- Manufacturer CFS300B datasheet, indexed at https://www.queclink.com/wp-content/uploads/2025/10/ENCFS300B-datasheet-250818.pdf — BLE 5.4, iBeacon broadcasting, adjustable broadcast interval (default 1s). BLE packet reception is not the same as a persistent paired connection. No CFS300B-to-this-firmware compatibility table or field-level command specification was verified.
- Queclink @Track Protocol Pro: https://www.queclink.com/development-integration-detail/track-protocol-pro — SDK, transport/QoS and TLS capabilities, but applicability to this existing device firmware is unverified.

## Existing Pastrack implementation

Read-only server inspection found the Go receiver owns TCP port 5001 and its health endpoint 18087. The web application runs on 18089. Local receiver connection handlers read incoming frames and spool them, without an IMEI-addressable command/session broker. Python receiver source also exists; do not switch receivers or disrupt telemetry merely to add a web page. Current web online status means a report was received within 5 minutes, not confirmed bidirectional command reachability. Last GPS fix time is a separate condition.

## Proposed integration

Device establishes the cellular connection to the configured Pastrack receiver; the web app does not dial the SIM's private carrier IP. Receiver associates a verified frame's IMEI with its current session, processes heartbeat/ACK as specified by the exact manual, and exposes a private command broker. Web submits validated named operations to a durable queue through an authenticated internal API or Unix socket. No arbitrary public AT console.

Presence states: connected session; recently reporting; stale heartbeat/report; offline/unknown; GPS freshness separately. Offline thresholds follow configured heartbeat/reporting/sleep profile instead of treating all devices with a single fixed threshold.

Command lifecycle: queued, expired-before-send, sent, acknowledged/rejected, verified-applied, response-timeout/unknown. Socket write success and protocol ACK must not be presented as proof of applied configuration. Match command sequence and IMEI, serialize per device, keep payload/secret redacted audit records, and read back settings where the protocol supports it. Do not automatically replay already-sent non-idempotent commands after an uncertain result. Unsent offline configuration can expire or be cancelled; restart/output actions require deliberate re-issue.

UI category Device management:
- Connection: session, transport, last packet, heartbeat, GPS freshness.
- Configuration: current and desired intervals, ignition/sleep profiles and other model-supported fields, validation, diff, apply/read-back.
- Accessories: model/MAC, supported accessory profile, scan/report diagnostics, last sensor reading, parsing/calibration status. For broadcast CFS300B, distinguish broadcasts seen, recognized/decoded reports, and stale/missing sensor data.
- Remote operations: supported diagnostics/status/location requests; reboot/output/firmware only after explicit model-specific support is verified. No documented SSH/RDP or remote-desktop capability was found for this tracker.
- Command history: actor, requested change, queue/session timing, response correlation and verification outcome.

## Required before enabling device commands

Obtain the manufacturer's GV30CEU @Track Air Interface Protocol matching protocol/firmware, including transport, heartbeat/ACK, command authentication, checksum/sequence, configuration read/write, BLE accessory commands and supported sensor matrix. Earlier screenshots show GTFRI/GTBTS/GTBAS/GTRTO interfaces; their exact network-wire field order and supported read-back semantics have not been verified. Do not copy parameter lists from another Queclink model. Obtain sensor integration instructions for CFS300B and confirm the actual firmware/password privately. Changing interval or accessory configuration is not authorized merely by collecting telemetry.

First device test should be a documented read-only status/configuration query on one user-authorized tracker. Configuration writes follow after response correlation is demonstrated, with an audit trail and preserved prior settings. No device commands were sent during this assessment and no receiver or live configuration was changed.
