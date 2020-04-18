#!/usr/bin/env python
import time
import requests

NAP_LENGTH = 0.2


def data_or_error(response):
    """Returns a tuple of (data, error).

    Data empty on error, error None if ok."""
    if response.status_code != requests.codes.ok:
        return {}, f'Got bad status code in response: {response.status_code}'

    try:
        response_wrapper = response.json()
    except ValueError as err:
        return {}, f'Could not parse json from response: {err}'

    try:
        # Data exists
        data = response_wrapper['data']
        # Don't care about totalCount key unless pagination becomes an issue
        # Be sure there's an error
        error = response_wrapper['error']
        # Be sure success is reported
        success = response_wrapper['success']
    except KeyError as err:
        return {}, f'Invalid response (missing parameter): {err}'

    if success != 'true':
        return {}, f'Request unsuccessful: {error}'

    if error != '':
        return {}, f'Request successful but error nonempty: {error}'

    if len(data) == 0:
        msg = 'Request successful but no data because JailTracker is garbage.'
        return {}, msg

    return data, None


class Jail:
    def __init__(self, jail_url):
        self.url = jail_url
        if jail_url[-1] != '/':
            self.url += '/'

    def get_inmates(self, limit=1000):
        # GET JailTracker/GetInmates?&start=0&limit=1000&sort=LastName&dir=ASC
        url = self.url + 'GetInmates'
        params = {'start': 0, 'limit': limit, 'sort': 'LastName', 'dir': 'ASC'}
        response = requests.get(url, params=params)
        return data_or_error(response)

    def get_inmate(self, arrest_no):
        # GET JailTracker/GetInmate?_dc=1576355374388&arrestNo=XXXXXXXXXX
        # I don't think the _dc arg matters
        url = self.url + 'GetInmate'
        params = {'arrestNo': arrest_no}
        response = requests.get(url, params=params)

        data, err = data_or_error(response)
        if err is None:
            # Inmate data format is ridiculous.
            data = {d['Field']: d['Value'] for d in data}
        return data, err

    def get_cases(self, arrest_no):
        # GET JailTracker/GetCases?arrestNo=XXXXXXXXXX
        url = self.url + 'GetCases'
        params = {'arrestNo': arrest_no}
        response = requests.get(url, params=params)
        return data_or_error(response)

    def get_charges(self, arrest_no):
        # POST JailTracker/GetCharges
        # Form data:  "arrestNo=XXXXXXXXXX"
        url = self.url + 'GetCharges'
        form_data = {'arrestNo': arrest_no}
        response = requests.post(url, data=form_data)
        return data_or_error(response)

    def process_inmate(self, arrest_no):
        inmate, err = self.get_inmate(arrest_no)
        if err is not None:
            msg = f'Skipping {arrest_no}. Could not get inmate data: {err}'
            return {}, msg

        cases, err = self.get_cases(arrest_no)
        if err is not None:
            print(f'Could not get case data for {arrest_no}: {err}')

        charges, err = self.get_charges(arrest_no)
        if err is not None:
            print(f'Could not get charge data for {arrest_no}: {err}')

        data = {'inmate': inmate,
                'cases': cases,
                'charges': charges}
        return data, None

    def process_inmates(self, limit=1000):
        inmates, err = self.get_inmates(limit)
        if err is not None:
            return {}, f'Could not get inmates: {err}'

        complete = []
        for summary in inmates:
            inmate, err = self.process_inmate(summary['ArrestNo'])
            if err is not None:
                print(err)
                continue
            inmate['Summary'] = summary
            complete.append(inmate)
            time.sleep(NAP_LENGTH)

        return complete, None
