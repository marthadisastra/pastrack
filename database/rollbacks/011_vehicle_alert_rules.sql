-- Development rollback only. Preserve alert history before removing definitions.
DELETE FROM alert_rules WHERE code IN ('FUEL_LOW','FUEL_DROP','OVER_SPEED');
DROP INDEX uq_alert_rules_code ON alert_rules;
ALTER TABLE alert_rules DROP COLUMN code;
