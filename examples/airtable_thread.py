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
import os
import threading

from jtt import jailtracker, storage

base_key = os.environ['AIRTABLE_BASE_KEY']
api_key = os.environ['AIRTABLE_API_KEY']

def main():
    with open('jails.json', 'r') as f:
        jails = json.load(f)

    # Kick off Airtable worker, while making JailTracker requests from main thread.
    airtable = storage.AirtableWriter(base_key, api_key)
    worker = storage.Worker(airtable.handle_intake)
    # Only one thread due to Airtable rate limiting
    threading.Thread(target=worker.run).start()

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

        # Queue these up to be pushed to Airtable while making more requests
        for inmate in inmates:
            worker.put(inmate)

    # Wait for queue to resolve
    worker.join()

if __name__ == '__main__':
    main()
