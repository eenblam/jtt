"""
Proof-of-concept Python module for working with the JailTracker API.
"""

import os
import sys
from dataclasses import dataclass
from typing import Any

import requests
from openai import OpenAI


openai_api_key = os.getenv('OPENAI_API_KEY')

openai_client = OpenAI(api_key=openai_api_key)

MAX_CAPTCHA_ATTEMPTS = 5
# This actually changes sometimes! e.g. https://omsweb.secure-gps.com
OMS_URL = 'https://omsweb.public-safety-cloud.com'
# Yes, "captcha" and "Captcha", as seen in the application traffic
GET_CAPTCHA_CLIENT_URL = f'{OMS_URL}/jtclientweb/captcha/getnewcaptchaclient'
VALIDATE_CAPTCHA_URL = f'{OMS_URL}/jtclientweb/Captcha/validatecaptcha'
# This key (like others) is typo'd. Using a variable here so it's easy to update if they fix the typo.
CAPTCHA_REQUIRED_KEY = 'captchaRequred'

def err(*args, **kwargs) -> None:
    """Wrapper for printing to stderr"""
    print(*args, file=sys.stderr, **kwargs)

@dataclass
class Jail:
    """
    Encapsulates both a jail's list of inmate data and the cross-request state needed to interact with the jail's API.

    At time of writing, the JailTracker API requires:

    * a captcha to be solved before initially requesting data (each captcha corresponds to a "captcha key")
    * an additional captcha to be solved after every 5th request for data
    * an "offender view key" that's updated with each request for data.
    """
    # Name of the jail, as it appears in the URL
    name: str
    session: requests.Session
    # Captcha key for this jail; updates ... sometimes? But has to be provided per-request.
    captcha_key: str
    #TODO improve typing here
    #TODO consider using our own terminology here. "Offenders" is what JT uses, but it's stupid.
    offenders: dict[str, Any]
    # Each request after validation updates this key!
    offender_view_key: str

    @classmethod
    def connect(cls, jail_name: str) -> 'Jail':
        """
        Solves an initial captcha, gets view key, and requests initial data for the given jail name.
        """
        session = requests.Session()
        session.headers.update({
            'Accept': '*/*',
            'Accept-Encoding': 'gzip, deflate, br',
            'Accept-Language': 'en-US,en;q=0.5',
            'Connection': 'keep-alive',
            'Origin': 'https://omsweb.public-safety-cloud.com',
            'User-Agent': 'Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:75.0) Gecko/20100101 Firefox/75.0',
            'X-Requested-With': 'XMLHttpRequest',
            'content-type': 'application/json; charset=utf-8'
            })

        jail = Jail(
                name=jail_name,
                session=session,
                offender_view_key='',
                offenders={},
                captcha_key='',
                )
        captcha_key = jail._process_captcha()
        api_url = get_jail_api_url(jail_name)

        # POST /jtclientweb/Offender/HANCOCK_COUNTY_MS with new captchaKey
        data = {
                'captchaKey': captcha_key,
                'captchaImage': None,
                'userCode': '',
        }
        resp = jail.session.post(api_url, json=data)
        resp_data = resp.json()
        try:
            captcha_required = resp_data[CAPTCHA_REQUIRED_KEY]
            jail.captcha_key = resp_data['captchaKey']
            offenders = resp_data['offenders']
            # (each request updates this!)
            jail.offender_view_key = resp_data['offenderViewKey']
        except KeyError as e:
            err(resp_data)
            raise e

        if captcha_required:
            raise RuntimeError('Captcha required for initial request')

        jail.offenders = {o['arrestNo']: o for o in offenders}

        return jail

    def get_inmate(self, arrest_no: str) -> dict:
        """
        Get inmate's info for the given arrest number.

        arrest_no should come from the original "offenders" list (o['arrestNo']).

        Note that an arrest number does not uniquely identify an inmate, since someone could be arrested multiple times.
        """
        #TODO this could use a better return type once a dataclass is implemented
        for _ in (1, 2):  # Extra attempt in case we need to process captcha
            # /jtclientweb/Offender/HANCOCK_COUNTY_MS/49949/offenderbucket/939027534
            location = f'{OMS_URL}/jtclientweb/Offender/{self.name}/{arrest_no}/offenderbucket/{self.offender_view_key}'
            data = {
                    'captchaKey': self.captcha_key,
                    'captchaImage': None,  # May need this?
                    'userCode': '',
            }
            resp = self.session.post(location, json=data)
            resp_data = resp.json()
            try:
                #captcha_required = resp_data['captchaRequired']
                # Yes, really. Lots of typos in their API.
                captcha_required = resp_data[CAPTCHA_REQUIRED_KEY]
                self.captcha_key = resp_data['captchaKey']
                self.offender_view_key = resp_data['offenderViewKey']
            except KeyError as e:
                print(resp_data, file=sys.stderr)
                raise e

            if not captcha_required:
                return resp_data

            print(f'Captcha required for arrest number {arrest_no}')
            self._process_captcha() 

        raise RuntimeError(f'Could not get inmate data for arrest number {arrest_no}')

    def update_inmate(self, arrest_no: str) -> dict:
        """
        Update inmate's info for the given arrest number.

        arrest_no should come from the original "offenders" list (o['arrestNo']).
        """
        inmate = self.get_inmate(arrest_no)

        o = self.offenders[arrest_no]
        if not o['cases']:
            o['cases'] = inmate['cases']
        if not o['charges']:
            o['charges'] = inmate['charges']

        return inmate

    def _process_captcha(self) -> str:
        """
        Process the captcha for the given jail name, returning captcha key on success.
        """
        # Referer should be the jail's URL; used for redirection in web client.
        # May not affect us, but matches "normal" traffic.
        headers = {'Referer': get_jail_url(self.name)}
        captcha_matched = False

        for i in range(MAX_CAPTCHA_ATTEMPTS):
            resp = self.session.get(GET_CAPTCHA_CLIENT_URL, headers=headers)
            resp_data = resp.json()
            # Response: {"captchaKey":"BASE64","captchaImage":"data:image/gif;base64,...BASE64...","userCode":null}
            try:
                captcha_key = resp_data['captchaKey']
                captcha_image = resp_data['captchaImage']
            except KeyError as e:
                err(e)
                raise e

            #text = extract_text_from_inline_image(captcha_image)
            text = solve_captcha_openai(captcha_image)
            print(f'Extracted text "{text}"')

            # Attempt validation
            data = {
                    'captchaKey': captcha_key,
                    'captchaImage': None,
                    'userCode': text,
                    }
            resp = self.session.post(VALIDATE_CAPTCHA_URL, json=data, headers=headers)
            resp_data = resp.json()
            # Error: {"captchaMatched": False, "captchaKey": "BASE64"}
            # Success: {"captchaMatched": True, "captchaKey": "NEW\BASE64=="}
            try:
                captcha_key = resp_data['captchaKey']
                captcha_matched = resp_data['captchaMatched']
            except KeyError as e:
                err(f'Missing key in response. Sent:\n{data}\n\Got:\n{resp_data}')
                raise e

            if captcha_matched:
                print(f'Matched captcha "{text}" and got captcha key "{captcha_key}"')
                self.captcha_key = captcha_key
                return captcha_key
            err(f'({i}/{MAX_CAPTCHA_ATTEMPTS}) Failed to match captcha. Detected "{text}".')

        raise RuntimeError(f'Failed to validate captcha after {MAX_CAPTCHA_ATTEMPTS} attempts')

def get_jail_url(name: str) -> str:
    # Not sure this is needed
    return f'https://omsweb.public-safety-cloud.com/jtclientweb/jailtracker/index/{name}'

def get_jail_api_url(name: str) -> str:
    return f'https://omsweb.public-safety-cloud.com/jtclientweb/Offender/{name}'


def solve_captcha_openai(inline_image: str) -> str:
    response = openai_client.chat.completions.create(
        model="gpt-4o",
        messages=[
            {
            "role": "system",
            "content": [
                {"type": "text", "text": "The user will send you images containing a single word of obfuscated text. Reply only with the text in the image, with no spaces or quotes."},
            ],
            },
            {
            "role": "user",
            "content": [
                {
                "type": "image_url",
                "image_url": {
                    "url": inline_image,
                },
                },
            ],
            }
        ],
        max_tokens=300,
    )

    return response.choices[0].message.content.strip()
