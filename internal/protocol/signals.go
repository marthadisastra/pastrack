package protocol

type SignalDefinition struct {
	MessageType, DisplayName, Category, Description, EventClass string
	SupportedDevice, FirmwareRequirement, ParserStatus          string
	EvaluatorStatus, AlertEligibility, DocumentationReference   string
}

var GV30CEUSignals = []SignalDefinition{
	{"GTCRA", "Crash Alarm", "CRASH", "Native crash alarm", "CRITICAL", "GV30CEU", "Protocol v0306; live 80201E0306 verified", "VERIFIED_LIVE", "TRIGGER", "CRASH_PRIMARY", "GV30CEU v0306 sec. 4.5.24"},
	{"GTECR", "Extended Crash Alarm", "CRASH", "Extended native crash alarm", "CRITICAL", "GV30CEU", "Protocol v0306; no live sample", "VERIFIED_DOCUMENT_ONLY", "CONDITIONAL_TRIGGER", "CRASH_PRIMARY_CONDITIONAL", "GV30CEU v0306 sec. 4.5.33"},
	{"GTCRD", "Crash Data", "CRASH", "Framed tri-axial crash reconstruction data", "SUPPORTING_EVIDENCE", "GV30CEU", "Protocol v0306; live 80201E0306 verified", "VERIFIED_LIVE", "SUPPORTING", "NO_NEW_ALERT", "GV30CEU v0306 sec. 4.6.1"},
	{"GTACC", "Acceleration Data", "TELEMETRY", "Continuous tri-axial acceleration packet", "TELEMETRY", "GV30CEU", "Protocol v0306; live 80201E0306 verified", "VERIFIED_LIVE", "INFORMATIONAL", "NO_ALERT", "GV30CEU v0306 sec. 4.6.2"},
	{"GTHBM", "Harsh Behavior", "DRIVING_BEHAVIOR", "Harsh braking, acceleration, or cornering report", "WARNING", "GV30CEU", "Protocol v0306", "CATALOG_ONLY", "NOT_IMPLEMENTED", "FUTURE_RULE", "GV30CEU v0306 sec. 4.2.1"},
	{"GTSPD", "Speed Alarm", "DRIVING_BEHAVIOR", "Device-native speed range alarm", "WARNING", "GV30CEU", "Protocol v0306", "CATALOG_ONLY", "NOT_IMPLEMENTED", "FUTURE_RULE", "GV30CEU v0306 sec. 4.2.1"},
	{"GTTOW", "Tow Alarm", "VEHICLE_SECURITY", "Motion-sensor tow alarm", "WARNING", "GV30CEU", "Protocol v0306", "CATALOG_ONLY", "NOT_IMPLEMENTED", "FUTURE_RULE", "GV30CEU v0306 sec. 4.2.1"},
	{"GTSOS", "SOS Alarm", "VEHICLE_SECURITY", "Configured digital input SOS report", "CRITICAL", "GV30CEU", "Protocol v0306; input configuration required", "CATALOG_ONLY", "NOT_IMPLEMENTED", "FUTURE_RULE", "GV30CEU v0306 sec. 4.2.1"},
	{"GTDIS", "Digital Input", "DEVICE_INPUT", "Digital input status change", "INFORMATIONAL", "GV30CEU", "Protocol v0306; physical wiring/configuration required", "CATALOG_ONLY", "NOT_IMPLEMENTED", "NO_ALERT", "GV30CEU v0306 sec. 4.2.1"},
	{"GTVGL", "Virtual Ignition", "DEVICE_INPUT", "Virtual ignition location report", "INFORMATIONAL", "GV30CEU", "Protocol v0306", "VERIFIED_PARSER", "NOT_ELIGIBLE", "NO_IGNITION_ALERT", "GV30CEU v0306 sec. 4.2.1"},
	{"GTIGL", "Ignition Location", "DEVICE_INPUT", "Physical ignition input location report", "INFORMATIONAL", "GV30CEU", "Protocol v0306; wiring required", "VERIFIED_LIVE", "NOT_ELIGIBLE", "NO_IGNITION_ALERT", "GV30CEU v0306 sec. 4.2.1"},
}
