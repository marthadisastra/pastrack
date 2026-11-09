import datetime as dt
import importlib.util
import math
from pathlib import Path
import unittest

spec=importlib.util.spec_from_file_location('simulation',Path(__file__).parents[1]/'deploy'/'simulate-october.py')
sim=importlib.util.module_from_spec(spec);spec.loader.exec_module(sim)

class SimulationTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.start=dt.date(2026,10,1)
        cls.until=dt.datetime(2026,10,30,3,20,tzinfo=dt.timezone.utc)
        cls.data=sim.build_dataset(cls.start,cls.until)
    def test_month_coverage_and_no_future_records(self):
        self.assertEqual(len(self.data),6)
        for car in self.data:
            self.assertEqual(len(car['days']),30)
            dates={p['at'].astimezone(sim.WIB).date() for p in car['points']}
            self.assertEqual(dates,{self.start+dt.timedelta(days=i) for i in range(30)})
            self.assertLessEqual(max(p['at'] for p in car['points']),self.until-dt.timedelta(minutes=1))
            self.assertLessEqual(len(car['points']),20000)
        self.assertLess(sum(len(c['points']) for c in self.data),49000)
    def test_varied_speeds_stops_heading_and_daily_scenarios(self):
        for car in self.data:
            points=car['points'];speeds={p['speed'] for p in points}
            self.assertGreater(len(speeds),100)
            self.assertIn(0,speeds)
            self.assertTrue(all(math.isfinite(s) and 0<=s<=112 for s in speeds))
            self.assertTrue(all(0<=p['heading']<360 for p in points))
            self.assertGreaterEqual(len({d['condition'] for d in car['days']}),2)
            for p in points:self.assertIn(p['state'],['parked','loading','moving','delivery_stop','traffic_stop','unloading'])
        self.assertTrue(any(d['parked'] for d in self.data[4]['days']))
        self.assertTrue(any(not d['parked'] for d in self.data[4]['days']))
        highway=max(p['speed'] for p in self.data[1]['points'])
        courier=max(p['speed'] for p in self.data[2]['points'])
        self.assertGreater(highway,90);self.assertLess(courier,40)
    def test_speed_matches_gps_and_no_cross_day_overlap(self):
        for car in self.data:
            for a,b in zip(car['points'],car['points'][1:]):
                seconds=(b['at']-a['at']).total_seconds()
                self.assertGreaterEqual(seconds,0)
                if seconds==0:
                    self.assertEqual(car['index'],5)
                    self.assertEqual((a['lat'],a['lon'],a['speed']),(b['lat'],b['lon'],b['speed']))
                if seconds==sim.STEP:
                    implied=sim.distance((a['lat'],a['lon']),(b['lat'],b['lon']))*3600/seconds
                    self.assertLess(abs(implied-b['speed']),.03)
    def test_reproducible_and_midnight_shift(self):
        a,points=sim.generate_day(3,dt.date(2026,10,12));b,again=sim.generate_day(3,dt.date(2026,10,12))
        self.assertEqual(points,again);self.assertEqual(a,b)
        self.assertGreater(len({p['at'].date() for p in points}),1)
        self.assertNotEqual(sim.daily_scenario(0,dt.date(2026,10,1)),sim.daily_scenario(0,dt.date(2026,10,2)))

if __name__=='__main__':unittest.main()
