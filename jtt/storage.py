#!/usr/bin/env python3

import json
import queue
from time import sleep

from airtable import Airtable
import requests

def nap():
    # We're allowed 5 requests / second
    # Assume each request takes 0.1, then sleep for the same
    sleep(0.1)

def push_to_table(table, record):
    # Note that rate-limiting is hard to detect here.
    # _process_response() in airtable-python-wrapper coerces the error to string
    # so we'd have to parse that to get the response code. :\
    # There's a semi-active issue here: https://github.com/gtalarico/airtable-python-wrapper/issues/83

    match = table.match('jtt_id', record['jtt_id'])
    if len(match) == 0:
        try:
            table.insert(record, typecast=True)
        except requests.exceptions.HTTPError as e:
            print(f'Request error: {e}')
    else:
        # Copies input dict without any fields that are None or empty string.
        # This prevents Airtable().update() from erasing previously collected data
        # that might have since been redacted.
        cleaned = {k:v for k,v in record.items() if v != "" and v is not None}
        try:
            table.update(match['id'], cleaned, typecast=True)
        except requests.exceptions.HTTPError as e:
            print(f'Request error: {e}')

class AirtableWriter(object):
    def __init__(self, base_key, api_key):
        self.intakes  = Airtable(base_key=base_key,
                                  table_name='intakes',
                                  api_key=api_key)

        self.cases = Airtable(base_key=base_key,
                                table_name='cases',
                                api_key=api_key)

        self.charges = Airtable(base_key=base_key,
                                  table_name='charges',
                                  api_key=api_key)

    def push_intake(self, intake):
        push_to_table(self.intakes, intake)

    def push_case(self, case):
        push_to_table(self.cases, case)

    def push_charge(self, charge):
        push_to_table(self.charges, charge)

    def handle_intake(self, result):
        intake = result['inmate']
        #HMM why is the value a list instead of just a string
        #intake_id = {'intake': [ intake['jtt_id'] ] }
        intake_id = intake['jtt_id']

        self.push_intake(intake)
        nap()
        for case in result['cases']:
            #case.update(intake_id)
            case['intake'] = intake_id
            self.push_case(case)
            nap()
        for charge in result['charges']:
            #charge.update(intake_id)
            charge['intake'] = intake_id
            self.push_charge(charge)
            nap()


class JsonWriter(object):
    """Just an example handler that doesn't require an airtable account"""
    def __init__(self, filename):
        self.filename = filename
        self.data = []

    def handle_intake(self, intake):
        self.data.append(intake)

    def write(self):
        with open(filename, 'w', encoding='utf-8') as f:
            json.dump(self.data, f, ensure_ascii=False, indent=4)


#HMM should other classes just inherit from worker instead of passing a method as a callback?
# Seems like a better design, but kinda erodes separation of concerns.
class Worker(object):
    def __init__(self, callback):
        self.queue = queue.Queue()
        self.callback = callback

    def run(self):
        while True:
            item = self.queue.get()
            self.callback(item)
            self.queue.task_done()

    def join(self):
        self.queue.join()

    def put(self, item):
        self.queue.put(item)
