"""Server-only pre-cutover check: disk queue survives DB outage and restart.
Uses existing .env through the shell; test rows are deleted after verification.
"""
import json
import os
import signal
import socket
import subprocess
import time
import urllib.error
import urllib.request

import mysql.connector

root = '/opt/pastrack'
old_pid = int(open(root + '/stage.pid').read())
os.kill(old_pid, signal.SIGTERM)
time.sleep(1)
env = os.environ.copy()
env.update(GPS_HOST='127.0.0.1', GPS_PORT='15001', HEALTH_ADDR='127.0.0.1:18088', SPOOL_DIR=root+'/stage-spool')
log = open(root+'/stage.log', 'ab')

def launch(bad=False):
    current = env.copy()
    if bad:
        current['DB_PORT'] = '13307'
    p = subprocess.Popen([root+'/pastrack'], env=current, stdout=log, stderr=log, user='www', group='www')
    time.sleep(1)
    assert p.poll() is None
    return p

p = launch(True)
try:
    urllib.request.urlopen('http://127.0.0.1:18088/healthz')
    raise AssertionError('health must detect DB outage')
except urllib.error.HTTPError as e:
    assert e.code == 503
sample = '+RESP:GTSTT,80201E0100,000000000000000,GV30CEU,41,0,0.6,64,24.9,106.872516,-6.133900,20261028061901,0510,0001,7D06,0337D73E,01,7,20261028100958,FFFF$'
unknown = '+RESP:GTXXX,80201E0100,000000000000000,GV30CEU,CFS300B,verification-only$'
malformed = '+RESP:GTSTT,80201E0100,000000000000000,GV30CEU$'
with socket.create_connection(('127.0.0.1',15001)) as s:
    s.sendall(sample[:19].encode())
    time.sleep(.1)
    s.sendall((sample[19:]+unknown+malformed).encode())
time.sleep(2)
assert len([n for n in os.listdir(env['SPOOL_DIR']) if n.endswith('.json')]) == 3
p.terminate();p.wait(timeout=10)
p = launch()
try:
    for _ in range(20):
        health = json.load(urllib.request.urlopen('http://127.0.0.1:18088/healthz'))
        if health['queued'] == 0:
            break
        time.sleep(1)
    assert health['queued'] == 0, health
    db = mysql.connector.connect(host=env['DB_HOST'], port=int(env['DB_PORT']), user=env['DB_USER'], password=env['DB_PASSWORD'], database=env['DB_NAME'])
    q = db.cursor()
    payloads = [sample, unknown, malformed]
    q.execute('SELECT raw_payload,parse_status FROM device_raw_messages WHERE imei=%s AND raw_payload IN (%s,%s,%s)', ('000000000000000', *payloads))
    rows = dict(q.fetchall())
    assert rows == {sample:'partial', unknown:'partial', malformed:'failed'}, rows
    q.execute('DELETE FROM device_raw_messages WHERE imei=%s AND raw_payload IN (%s,%s,%s)', ('000000000000000', *payloads))
    db.commit();db.close()
    print('PASS: fragmentation/coalescing, DB outage, durable queue, restart recovery, unknown fuel raw fields, malformed frame storage; test rows cleaned')
finally:
    p.terminate();p.wait(timeout=10)
