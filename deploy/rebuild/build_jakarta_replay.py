"""Build deterministic replay samples from a road route with three legs."""
import bisect
import json
import math
import sys
import urllib.request
from datetime import datetime, timedelta
from pathlib import Path

WAYPOINTS = [(106.8722407, -6.1331672), (106.7321071, -6.2079007), (106.6518227, -6.224809)]
SCENARIOS = [
    ('2026-10-29', 'normal', (45, 25, 95), 0),
    ('2026-10-30', 'macet', (75, 40, 145), -40),
    ('2026-10-31', 'lancar', (30, 15, 65), 60),
]

def distance(a, b):
    x, y, u, v = map(math.radians, (*a, *b))
    h = math.sin((y-v)/2)**2 + math.cos(y)*math.cos(v)*math.sin((x-u)/2)**2
    return 6371*2*math.asin(min(1, math.sqrt(h)))

def route_legs(destination):
    points = WAYPOINTS + [destination]
    legs = []
    for a, b in zip(points, points[1:]):
        query = ';'.join(f'{lon},{lat}' for lon, lat in (a,b))
        with urllib.request.urlopen(f'https://router.project-osrm.org/route/v1/driving/{query}?overview=full&geometries=geojson', timeout=35) as response:
            result = json.load(response)
        coords = result['routes'][0]['geometry']['coordinates']
        assert distance(a, coords[0]) < .25 and distance(b, coords[-1]) < .25
        legs.append(coords)
    return legs

def build(legs):
    output = []
    for day, traffic, minutes, delta in SCENARIOS:
        start = datetime.fromisoformat(day+'T08:00:00') - timedelta(hours=7)
        samples, trips = [], []
        traveled = consumed = adjustment = 0
        event = None
        def add(point, speed):
            samples.append(dict(at=(start+timedelta(seconds=15*len(samples))).isoformat(),
                longitude=point[0], latitude=point[1], speed=round(speed,2),
                liters=round(220-consumed+adjustment,3)))
        add(legs[0][0],0)
        for index, (coords, duration) in enumerate(zip(legs, minutes)):
            cumulative = [0.0]
            for a,b in zip(coords,coords[1:]):
                cumulative.append(cumulative[-1]+distance(a,b))
            steps=duration*4
            weights=[(0 if traffic=='macet' and i%16<4 else 1)*(1+.38*math.sin(i*.11)) for i in range(steps)]
            total=sum(weights)
            first=len(samples)
            offset=0.0
            for weight in weights:
                km=cumulative[-1]*weight/total
                offset=min(cumulative[-1],offset+km)
                k=min(len(coords)-1,max(1,bisect.bisect_left(cumulative,offset)))
                f=(offset-cumulative[k-1])/max(1e-9,cumulative[k]-cumulative[k-1])
                point=[a+(b-a)*f for a,b in zip(coords[k-1],coords[k])]
                traveled+=km
                consumed+=km/(7 if traffic=='macet' else 8)
                add(point,km*240)
            last=len(samples)-1
            trips.append(dict(first=first,last=last,distance_km=cumulative[-1],
                seconds=duration*60))
            stop=24 if index<2 else 16
            event_start=len(samples)+3
            for tick in range(stop):
                if index==1 and delta and 4<=tick<=11:
                    adjustment+=delta/8
                add(coords[-1],0)
            if index==1 and delta:
                event=dict(first=event_start,last=event_start+8,delta=delta,
                    kind='fuel_drop' if delta<0 else 'refueling')
        assert all(0<=s['liters']<=400 and 0<=s['speed']<=130 for s in samples)
        if event:
            assert abs(samples[event['last']]['liters']-samples[event['first']]['liters']-delta)<.001
        output.append(dict(day=day,traffic=traffic,samples=samples,trips=trips,event=event,
            distance_km=round(traveled,3),consumed_l=round(consumed,3)))
    return output

if __name__=='__main__':
    lon,lat=float(sys.argv[1]),float(sys.argv[2])
    Path(sys.argv[3]).write_text(json.dumps(build(route_legs((lon,lat)))))
