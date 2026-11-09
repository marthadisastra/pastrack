-- Development rollback only; succeeds only when no CRASH alert history references the rule.
DELETE FROM alert_rules WHERE code='CRASH';
DROP TABLE IF EXISTS crash_signal_events;
DROP TABLE IF EXISTS crash_evidence_sets;
DROP TABLE IF EXISTS queclink_signal_catalog;
