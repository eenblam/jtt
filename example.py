#!/usr/bin/env python3

"""
This script assumes that you have some file `jails.json` with the following format:
{
  "Perry County": {
    "name": "Perry",  <--- Optional tag for logging. Make it whatever you want or omit entirely.
    "url": "https://omsweb.public-safety-cloud.com/jtclientweb/jailtracker/index/Perry_County_MS"
  },

  ...,

  "Yazoo County": {
    "name": "Yazoo",
    "url": "https://omsweb.public-safety-cloud.com/jtclientweb/jailtracker/index/Yazoo_County_MS"
  }
}

It aggregates all the data available for each jail, then writes it out to `results.json`.
"""

import json
import requests
import jailtracker

with open('jails.json', 'r') as f:
    jails = json.load(f)

data = {}

for county,vals in jails.items():
    print(f"Trying {county}...")

    try:
        name = vals['name']
    except KeyError:
        name = ''

    try:
        jail = jailtracker.Jail(vals['url'], name)
    except RuntimeError as err:
        print(f"Skipping {county}! Could not get session URL: {err}")
        continue

    inmates, err = jail.process_inmates()
    if err is not None:
        print(err)
        continue

    data[county] = inmates

with open('results.json', 'w', encoding='utf-8') as f:
    json.dump(data, f, ensure_ascii=False, indent=4)
