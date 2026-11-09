"""Remove the two known October fixture imports after validating their provenance."""
import json

def remove_october_fixture(connection, archive=None):
    cursor = connection.cursor()
    connection.begin()
    try:
        cursor.execute("""
            SELECT v.id, a.device_id FROM vehicles v JOIN vehicle_assignments a ON a.vehicle_id=v.id
            WHERE v.plate_number IN ('B 9101 PTR','B 9102 PTR') AND a.provenance='fixture_oct31'
            FOR UPDATE
        """)
        fixtures = cursor.fetchall()
        if not fixtures:
            connection.commit()
            return 0
        assert len(fixtures) == 2
        for row in fixtures:
            vehicle, device = row['id'], row['device_id']
            def count(sql, args):
                cursor.execute(sql, args)
                return cursor.fetchone()['n']
            assert count("SELECT COUNT(*) n FROM vehicle_assignments WHERE vehicle_id=%s OR device_id=%s", (vehicle, device)) == 1
            assert count("SELECT COUNT(*) n FROM device_raw_messages WHERE device_id=%s", (device,)) == 721
            assert count("SELECT COUNT(*) n FROM device_raw_messages WHERE device_id=%s AND JSON_VALID(CONVERT(payload USING utf8mb4)) AND JSON_UNQUOTE(JSON_EXTRACT(CONVERT(payload USING utf8mb4),'$.origin'))='fixture_oct31'", (device,)) == 721
            assert count("SELECT COUNT(*) n FROM gps_positions p JOIN device_raw_messages r ON r.id=p.raw_message_id WHERE p.vehicle_id=%s AND p.device_id=%s AND r.device_id=%s", (vehicle, device, device)) == 721
            assert count("SELECT COUNT(*) n FROM gps_positions WHERE vehicle_id=%s OR device_id=%s", (vehicle, device)) == 721
            assert count("SELECT COUNT(*) n FROM fuel_readings f JOIN sensor_telemetry t ON t.id=f.telemetry_id WHERE f.vehicle_id=%s AND t.device_id=%s AND t.decoder_version='fixture_oct31'", (vehicle, device)) == 721
            assert count("SELECT COUNT(*) n FROM fuel_readings WHERE vehicle_id=%s", (vehicle,)) == 721
            assert count("SELECT COUNT(*) n FROM trips WHERE vehicle_id=%s AND calculation_version<>'fixture_oct31'", (vehicle,)) == 0
            assert count("SELECT COUNT(*) n FROM fuel_events WHERE vehicle_id=%s AND rule_version<>'fixture_oct31'", (vehicle,)) == 0
            cursor.execute("SELECT f.id fuel_id,b.sensor_id FROM fuel_sensors f JOIN ble_sensor_bindings b ON b.sensor_id=f.sensor_id WHERE b.vehicle_id=%s AND b.device_id=%s AND b.binding_source='fixture_oct31'", (vehicle, device))
            sensors = cursor.fetchall()
            assert len(sensors) == 1
            fuel, sensor = sensors[0]['fuel_id'], sensors[0]['sensor_id']
            cursor.execute("SELECT * FROM fuel_event_investigations WHERE event_id IN (SELECT id FROM fuel_events WHERE vehicle_id=%s) FOR UPDATE", (vehicle,))
            investigations = cursor.fetchall()
            if investigations:
                assert archive is not None
                archive_file = archive.with_name(archive.name + '.' + str(vehicle))
                with archive_file.open('x', encoding='utf-8') as file:
                    archive_file.chmod(0o600)
                    json.dump(investigations, file, default=str)
                cursor.execute("DELETE FROM fuel_event_investigations WHERE event_id IN (SELECT id FROM fuel_events WHERE vehicle_id=%s)", (vehicle,))
            statements = [
                ("DELETE FROM fuel_events WHERE vehicle_id=%s AND rule_version='fixture_oct31'", vehicle),
                ("DELETE FROM fuel_consumption_summaries WHERE vehicle_id=%s AND algorithm_version='fixture_oct31'", vehicle),
                ("DELETE FROM trips WHERE vehicle_id=%s AND calculation_version='fixture_oct31'", vehicle),
                ("DELETE FROM latest_positions WHERE vehicle_id=%s", vehicle),
                ("DELETE FROM latest_fuel_readings WHERE fuel_sensor_id=%s", fuel),
                ("DELETE FROM ble_sensor_status WHERE sensor_id=%s", sensor),
                ("DELETE FROM gps_device_status WHERE device_id=%s", device),
                ("DELETE FROM device_events WHERE vehicle_id=%s AND source='fixture_oct31'", vehicle),
                ("DELETE FROM fuel_readings WHERE vehicle_id=%s", vehicle),
                ("DELETE FROM fuel_calibration_points WHERE calibration_id IN (SELECT id FROM fuel_calibrations WHERE fuel_sensor_id=%s) AND evidence='fixture_oct31'", fuel),
                ("DELETE FROM fuel_calibrations WHERE fuel_sensor_id=%s", fuel),
                ("DELETE FROM fuel_sensors WHERE id=%s", fuel),
                ("DELETE FROM fuel_tank_profiles WHERE vehicle_id=%s", vehicle),
                ("DELETE FROM sensor_telemetry WHERE device_id=%s AND decoder_version='fixture_oct31'", device),
                ("DELETE FROM ble_sensor_bindings WHERE sensor_id=%s AND binding_source='fixture_oct31'", sensor),
                ("DELETE FROM ble_sensors WHERE id=%s", sensor),
                ("DELETE FROM gps_positions WHERE vehicle_id=%s", vehicle),
                ("DELETE FROM protocol_parse_logs WHERE parser_version='fixture_oct31' AND raw_message_id IN (SELECT id FROM device_raw_messages WHERE device_id=%s)", device),
                ("DELETE FROM device_raw_messages WHERE device_id=%s", device),
                ("DELETE FROM vehicle_assignments WHERE vehicle_id=%s AND provenance='fixture_oct31'", vehicle),
                ("DELETE FROM gps_devices WHERE id=%s", device),
                ("DELETE FROM vehicle_assets WHERE vehicle_id=%s", vehicle),
                ("DELETE FROM vehicles WHERE id=%s", vehicle),
            ]
            for sql, key in statements:
                cursor.execute(sql, (key,))
        connection.commit()
        return len(fixtures)
    except BaseException:
        connection.rollback()
        raise
