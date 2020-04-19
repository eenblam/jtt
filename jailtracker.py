#!/usr/bin/env python

import re
import time
from  urllib.parse import urljoin

import requests

NAP_LENGTH = 0.2
LOCATION_PATTERN = re.compile('/jtclientweb/\(S\([a-zA-Z0-9]+\)\)/jailtracker/')
LOCATION_BASE_URL = 'https://omsweb.public-safety-cloud.com'


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
        self.session = requests.Session()
        self.session.headers.update({
            'Accept': '*/*',
            'Accept-Encoding': 'gzip, deflate, br',
            'Accept-Language': 'en-US,en;q=0.5',
            'Connection': 'keep-alive',
            'Origin': 'https://omsweb.public-safety-cloud.com',
            'User-Agent': 'Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:75.0) Gecko/20100101 Firefox/75.0',
            'X-Requested-With': 'XMLHttpRequest'
            })

        print(f"Getting session URL from {jail_url}...")
        err = self.set_session_info(jail_url)
        if err is not None:
            raise RuntimeError(err)


    def set_session_info(self, url):
        """Tries to set session info. Returns an error if unable to.

        JailTracker tracks sessions in their URIs.
        To get one, visit a jail's website, e.g.
        https://omsweb.public-safety-cloud.com/jtclientweb/jailtracker/index/HANCOCK_COUNTY_MS

        You should receive a 302 response, with a Location header like this:
        https://omsweb.public-safety-cloud.com/jtclientweb/(S(nejsvux3kuvd4lexebac2qia))/jailtracker/index/HANCOCK_COUNTY_MS

        Note that you need to first visit the provided Location before making any queries!

        After doing so, you can make get the API URI for querying by stripping `index/<county>`, e.g.
        https://omsweb.public-safety-cloud.com/jtclientweb/(S(nejsvux3kuvd4lexebac2qia))/jailtracker/
        """
        response = self.session.get(url, allow_redirects=False)
        if response.status_code != requests.codes.found:
            return 0, f'Expected 302 Found, got {response.status_code}'

        try:
            location = response.headers['Location']
        except KeyError:
            return '', f'Got 302 Found, but Location header not set'

        if location == '':
            return '', f'Location found but empty'

        # Okay, we found Location.
        # This will later be our Referer header, but we need to extract the API URL as well.
        match = LOCATION_PATTERN.match(location)
        if match is None:
            return f"Couldn't parse location {location} - no match found."

        try:
            target = match.group(0)
        except IndexError:
            return '', f'Expected match for {location} but found none'

        # We'll get weird 'session-time-out' responses if we never visit this page.
        # (Not a real timeout. We get a 200, but the `error` message is `session-time-out`.)
        knock_response = self.session.get(urljoin(LOCATION_BASE_URL, location))
        if knock_response.status_code != requests.codes.ok:
            print(f'WARN Got bad status code in knock response: {knock_response.status_code}')

        # All good.
        self.url = urljoin(LOCATION_BASE_URL, target)
        print(f'Using {self.url}.')

        self.session.headers.update({
            'Referer': location,
            })

        return None

    def get_inmates(self, limit=1000):
        # GET JailTracker/GetInmates?&start=0&limit=1000&sort=LastName&dir=ASC
        url = self.url + 'GetInmates'
        params = {'start': 0, 'limit': limit, 'sort': 'LastName', 'dir': 'ASC'}
        response = self.session.get(url, params=params)
        return data_or_error(response)

    def get_inmate(self, arrest_no):
        # GET JailTracker/GetInmate?_dc=1576355374388&arrestNo=XXXXXXXXXX
        # I don't think the _dc arg matters
        url = self.url + 'GetInmate'
        params = {'arrestNo': arrest_no}
        response = self.session.get(url, params=params)

        data, err = data_or_error(response)
        if err is None:
            # Inmate data format requires some cleaning..
            # Have: {..., {'Field': 'FieldName:', 'Value': 'some value'}, ...}
            # Want: {..., 'FieldName': 'some value', ...}
            data = {d['Field'].rstrip(':'): d['Value'] for d in data}
        return data, err

    def get_cases(self, arrest_no):
        # GET JailTracker/GetCases?arrestNo=XXXXXXXXXX
        url = self.url + 'GetCases'
        params = {'arrestNo': arrest_no}
        response = self.session.get(url, params=params)
        return data_or_error(response)

    def get_charges(self, arrest_no):
        # POST JailTracker/GetCharges
        # Form data:  "arrestNo=XXXXXXXXXX"
        url = self.url + 'GetCharges'
        form_data = {'arrestNo': arrest_no}
        response = self.session.post(url, data=form_data, headers={'Content-Type': 'application/x-www-form-urlencoded'})
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

        total = len(inmates)
        print(f'Processing data for {total} inmates.')
        complete = []
        for summary in inmates:
            inmate, err = self.process_inmate(summary['ArrestNo'])
            if err is not None:
                print(err)
                continue
            inmate['summary'] = summary
            complete.append(inmate)
            time.sleep(NAP_LENGTH)

        no_cases = len([x for x in complete if len(x['cases']) == 0])
        no_charges = len([x for x in complete if len(x['charges']) == 0])
        print(f'Number of inmates with no cases linked: {no_cases}/{total}')
        print(f'Number of inmates with no charges linked: {no_charges}/{total}')

        return complete, None
