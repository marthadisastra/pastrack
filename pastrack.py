
#!/usr/bin/env python3

import os
import json
import socket
import threading
import logging

from datetime import datetime, timezone
from zoneinfo import ZoneInfo

import mysql.connector
from mysql.connector import Error


# ============================================================
# SERVER CONFIGURATION
# ============================================================

HOST = os.getenv("GPS_HOST", "0.0.0.0")
PORT = int(os.getenv("GPS_PORT", "5001"))
BUFFER_SIZE = 8192
LOG_FILE = "queclink_log.txt"
SOCKET_TIMEOUT = 300

# ============================================================
# DATABASE CONFIGURATION
# ============================================================

DB_CONFIG = {
    "host": os.getenv("DB_HOST", "127.0.0.1"),
    "port": int(os.getenv("DB_PORT", "3306")),
    "database": os.getenv("DB_NAME", "gps_tracker_rnd"),
    "user": os.getenv("DB_USER", "gps_listener"),
    "password": os.getenv("DB_PASSWORD", ""),
    "connection_timeout": 10,
    "autocommit": False,
}

# Waktu GPS dari perangkat diasumsikan UTC.
# Ubah GPS_TIMEZONE jika dokumentasi perangkat menunjukkan
# bahwa timestamp payload menggunakan zona waktu lain.
GPS_TIMEZONE = ZoneInfo(os.getenv("GPS_TIMEZONE", "UTC"))


# ============================================================
# LOGGING
# ============================================================

logger = logging.getLogger("queclink")
logger.setLevel(logging.INFO)
logger.propagate = False

if not logger.handlers:
    formatter = logging.Formatter(
        "%(asctime)s,%(msecs)03d [%(levelname)s] %(message)s",
        datefmt="%Y-%m-%d %H:%M:%S",
    )

    console_handler = logging.StreamHandler()
    console_handler.setFormatter(formatter)
    logger.addHandler(console_handler)

    file_handler = logging.FileHandler(
        LOG_FILE,
        mode="a",
        encoding="utf-8",
    )
    file_handler.setFormatter(formatter)
    logger.addHandler(file_handler)


# ============================================================
# DATABASE HELPERS
# ============================================================

def get_db_connection():
    return mysql.connector.connect(**DB_CONFIG)


def utc_now_naive():
    """MySQL DATETIME menyimpan waktu UTC tanpa timezone."""
    return datetime.now(timezone.utc).replace(tzinfo=None)


def parse_device_datetime(value):
    """
    Format timestamp Queclink:
    YYYYMMDDHHMMSS

    Contoh:
    20261028061901

    Hasil disimpan sebagai UTC-naive DATETIME.
    """
    if not value or len(value) != 14 or not value.isdigit():
        return None

    dt = datetime.strptime(value, "%Y%m%d%H%M%S")
    dt = dt.replace(tzinfo=GPS_TIMEZONE)

    return dt.astimezone(timezone.utc).replace(tzinfo=None)


def get_device(cursor, imei):
    cursor.execute(
        """
        SELECT id
        FROM gps_devices
        WHERE imei = %s
        LIMIT 1
        """,
        (imei,),
    )

    row = cursor.fetchone()
    return row[0] if row else None


# ============================================================
# QUECLINK MESSAGE PARSER
# ============================================================

def parse_queclink_message(payload):
    """
    Parsing awal berdasarkan frame GTSTT yang diberikan.

    Jangan menganggap field status, akurasi, satelit,
    dan field tambahan telah terverifikasi tanpa manual
    protokol GV30CEU yang sesuai.
    """

    text = payload.strip()

    if text.endswith("$"):
        text = text[:-1]

    parts = text.split(",")

    if len(parts) < 4:
        raise ValueError("Payload terlalu pendek")

    header = parts[0]

    if not header.startswith("+RESP:"):
        raise ValueError(
            f"Header bukan response yang didukung: {header}"
        )

    message_type = header[len("+RESP:"):]

    protocol_version = parts[1]
    imei = parts[2]
    model = parts[3]

    result = {
        "message_type": message_type,
        "protocol_version": protocol_version,
        "imei": imei,
        "model": model,
        "fields": parts[4:],
    }

    # Parsing posisi GTSTT.
    # Indeks dihitung dari keseluruhan parts, termasuk header.
    if message_type == "GTSTT":
        if len(parts) < 12:
            raise ValueError("Field GTSTT tidak lengkap")

        result.update({
            "status_code_raw": parts[4],
            "accuracy_raw": parts[5],
            "speed_kmh": float(parts[6]),
            "heading_deg": int(parts[7]),
            "altitude_m": float(parts[8]),
            "longitude": float(parts[9]),
            "latitude": float(parts[10]),
            "gps_time": parse_device_datetime(parts[11]),
            "device_send_time":parse_device_datetime(parts[18]),
        })

        # Validasi koordinat sebelum disimpan sebagai posisi.
        if not -90 <= result["latitude"] <= 90:
            raise ValueError("Latitude di luar rentang valid")

        if not -180 <= result["longitude"] <= 180:
            raise ValueError("Longitude di luar rentang valid")

        if result["speed_kmh"] < 0:
            raise ValueError("Kecepatan tidak boleh negatif")

        if not 0 <= result["heading_deg"] <= 359:
            raise ValueError("Heading di luar rentang valid")

        # Nilai mentah jaringan dipertahankan.
        # Belum dikonversi menjadi integer agar tidak
        # kehilangan representasi HEX.
        if len(parts) > 15:
            result["network"] = {
                "mcc_raw": parts[12],
                "mnc_raw": parts[13],
                "lac_raw": parts[14],
                "cell_id_raw": parts[15],
            }

    return result


# ============================================================
# DATABASE INGESTION
# ============================================================

def save_message(payload, received_at):
    """
    Simpan raw message dan hasil parsing dalam satu transaksi.

    Raw payload tetap disimpan jika parsing gagal,
    selama database dapat diakses.
    """

    conn = None
    cursor = None

    imei = None
    message_type = None
    protocol_version = None
    parsed = None
    parse_error = None
    parse_status = "failed"

    try:
        parsed = parse_queclink_message(payload)

        imei = parsed.get("imei")
        message_type = parsed.get("message_type")
        protocol_version = parsed.get("protocol_version")

        parse_status = "success"

    except (ValueError, TypeError, IndexError, OverflowError) as exc:
        parse_error = str(exc)

        # Ambil identitas dasar jika header masih dapat dibaca.
        parts = payload.strip().rstrip("$").split(",")

        if len(parts) >= 4:
            header = parts[0]

            if header.startswith("+RESP:"):
                message_type = header[len("+RESP:"):]

            protocol_version = parts[1]
            imei = parts[2]

    try:
        conn = get_db_connection()
        cursor = conn.cursor()

        device_id = get_device(cursor, imei) if imei else None

        if parsed and device_id is None:
            parse_status = "partial"
            parse_error = (
                "IMEI belum terdaftar di gps_devices; "
                "raw payload tetap disimpan."
            )

        # 1. Simpan payload mentah terlebih dahulu.
        cursor.execute(
            """
            INSERT INTO device_raw_messages (
                device_id,
                imei,
                message_type,
                protocol_version,
                raw_payload,
                parsed_payload,
                received_at,
                parse_status,
                parse_error
            )
            VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s)
            """,
            (
                device_id,
                imei,
                message_type,
                protocol_version,
                payload,
                json.dumps(parsed, default=str) if parsed else None,
                received_at,
                "pending",
                None,
            ),
        )

        raw_message_id = cursor.lastrowid

        # 2. Simpan posisi dan event jika perangkat terdaftar.
        if parsed and device_id is not None:
            cursor.execute(
                """
                UPDATE gps_devices
                SET last_seen_at = %s
                WHERE id = %s
                """,
                (received_at, device_id),
            )

            if message_type == "GTSTT":
                cursor.execute(
                    """
                    INSERT INTO gps_positions (
                        device_id,
                        raw_message_id,
                        gps_time,
                        latitude,
                        longitude,
                        speed_kmh,
                        heading_deg,
                        altitude_m,
                        gps_accuracy,
                        satellites
                    )
                    VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
                    """,
                    (
                        device_id,
                        raw_message_id,
                        parsed.get("gps_time"),
                        parsed.get("latitude"),
                        parsed.get("longitude"),
                        parsed.get("speed_kmh"),
                        parsed.get("heading_deg"),
                        parsed.get("altitude_m"),
                        None,  # Belum diverifikasi
                        None,  # Belum diverifikasi
                    ),
                )

                cursor.execute(
                    """
                    INSERT INTO device_events (
                        device_id,
                        raw_message_id,
                        message_type,
                        event_code,
                        event_value,
                        event_time
                    )
                    VALUES (%s, %s, %s, %s, %s, %s)
                    """,
                    (
                        device_id,
                        raw_message_id,
                        message_type,
                        parsed.get("status_code_raw"),
                        "Unverified protocol status code",
                        parsed.get("gps_time"),
                    ),
                )

        # 3. Perbarui status parsing setelah pemrosesan.
        cursor.execute(
            """
            UPDATE device_raw_messages
            SET parse_status = %s,
                parse_error = %s
            WHERE id = %s
            """,
            (parse_status, parse_error, raw_message_id),
        )

        conn.commit()

        logger.info(
            "DATABASE SAVED | RAW_ID=%s | IMEI=%s | TYPE=%s | STATUS=%s",
            raw_message_id,
            imei,
            message_type,
            parse_status,
        )

        if parsed and message_type == "GTSTT" and device_id is not None:
            logger.info(
                "GPS POSITION | IMEI=%s | LAT=%s | LON=%s | "
                "SPEED=%s km/h | HEADING=%s",
                imei,
                parsed["latitude"],
                parsed["longitude"],
                parsed["speed_kmh"],
                parsed["heading_deg"],
            )

        if parse_error:
            logger.warning(
                "PARSE WARNING | IMEI=%s | ERROR=%s",
                imei,
                parse_error,
            )

    except Error:
        if conn is not None:
            conn.rollback()

        logger.exception(
            "DATABASE ERROR | IMEI=%s | TYPE=%s",
            imei,
            message_type,
        )

    except Exception:
        if conn is not None:
            conn.rollback()

        logger.exception("MESSAGE PROCESSING ERROR")

    finally:
        if cursor is not None:
            cursor.close()

        if conn is not None and conn.is_connected():
            conn.close()


# ============================================================
# TCP CLIENT HANDLER
# ============================================================

def handle_client(conn, addr):
    client_ip, client_port = addr

    logger.info(
        "TRACKER CONNECTED | IP=%s | PORT=%s",
        client_ip,
        client_port,
    )

    buffer = bytearray()

    try:
        with conn:
            conn.settimeout(SOCKET_TIMEOUT)

            while True:
                try:
                    data = conn.recv(BUFFER_SIZE)

                except socket.timeout:
                    logger.info(
                        "TIMEOUT | IP=%s | PORT=%s",
                        client_ip,
                        client_port,
                    )
                    break

                if not data:
                    logger.info(
                        "DISCONNECTED | IP=%s | PORT=%s",
                        client_ip,
                        client_port,
                    )
                    break

                logger.info(
                    "GPS DATA | IP=%s | BYTES=%d",
                    client_ip,
                    len(data),
                )

                logger.info("HEX: %s", data.hex(" "))

                logger.info(
                    "TEXT: %s",
                    data.decode("ascii", errors="replace"),
                )

                buffer.extend(data)

                # Setiap frame Queclink berakhir dengan '$'.
                while b"$" in buffer:
                    frame, _, remainder = buffer.partition(b"$")
                    buffer = bytearray(remainder)

                    frame = bytes(frame) + b"$"

                    if not frame.strip():
                        continue

                    received_at = utc_now_naive()

                    logger.info(
                        "FRAME RECEIVED | TIME_UTC=%s | IP=%s | SIZE=%d",
                        received_at.isoformat(timespec="milliseconds"),
                        client_ip,
                        len(frame),
                    )

                    try:
                        payload = frame.decode("ascii").strip()

                    except UnicodeDecodeError:
                        logger.exception(
                            "INVALID ASCII FRAME | IP=%s",
                            client_ip,
                        )
                        continue

                    save_message(payload, received_at)

                # Proteksi buffer jika terminator tidak kunjung datang.
                if len(buffer) > 1024 * 1024:
                    logger.error(
                        "FRAME BUFFER TOO LARGE | IP=%s | SIZE=%d",
                        client_ip,
                        len(buffer),
                    )
                    buffer.clear()

    except OSError as exc:
        logger.warning(
            "SOCKET ERROR | IP=%s | ERROR=%s",
            client_ip,
            exc,
        )

    finally:
        logger.info("CONNECTION CLOSED | IP=%s", client_ip)


# ============================================================
# MAIN SERVER
# ============================================================

def main():
    # Validasi koneksi database sebelum membuka listener.
    conn = get_db_connection()
    conn.close()

    logger.info("DATABASE CONNECTION OK")

    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as server:
        server.setsockopt(
            socket.SOL_SOCKET,
            socket.SO_REUSEADDR,
            1,
        )

        server.bind((HOST, PORT))
        server.listen(50)

        logger.info(
            "QUECLINK TCP SERVER LISTENING ON %s:%s",
            HOST,
            PORT,
        )
        logger.info("LOG FILE: %s", LOG_FILE)

        while True:
            try:
                conn, addr = server.accept()

                thread = threading.Thread(
                    target=handle_client,
                    args=(conn, addr),
                    daemon=True,
                )
                thread.start()

            except KeyboardInterrupt:
                logger.info("SERVER STOPPED")
                break

            except OSError:
                logger.exception("SERVER SOCKET ERROR")


if __name__ == "__main__":
    try:
        main()

    except KeyboardInterrupt:
        logger.info("SERVER STOPPED BY USER")

    except Exception:
        logger.exception("FATAL SERVER ERROR")

    finally:
        logging.shutdown()
