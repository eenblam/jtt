#!/usr/bin/env python3

from collections import defaultdict as dd
import json

with open('results.json', 'r') as f:
    data = json.load(f)

summary_keys = dd(set)
inmate_keys = dd(set)
case_keys = dd(set)
charge_keys = dd(set)

for county,inmates in data.items():
    for i in inmates:
        for key in i['summary'].keys():
            summary_keys[key].add(county)
        for key in i['inmate'].keys():
            inmate_keys[key].add(county)
        for case in i['cases']:
            for key in case.keys():
                case_keys[key].add(county)
        for charge in i['charges']:
            for key in charge.keys():
                charge_keys[key].add(county)

keys = {
        'summary': {k:list(v) for k,v in summary_keys.items()},
        'inmate': {k:list(v) for k,v in inmate_keys.items()},
        'case': {k:list(v) for k,v in case_keys.items()},
        'charge': {k:list(v) for k,v in charge_keys.items()}
        }

with open('column_results.json', 'w', encoding='utf-8') as f:
    json.dump(keys, f, ensure_ascii=False, indent=4)

totals = {k:list(v) for k,v in keys.items()}
with open('column_totals.json', 'w', encoding='utf-8') as f:
    json.dump(totals, f, ensure_ascii=False, indent=4)
