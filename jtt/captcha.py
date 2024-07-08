import base64
import os
import sys
from dataclasses import dataclass
from io import BytesIO
from typing import Any

import cv2
import numpy as np
import pytesseract
import requests
from openai import OpenAI
from PIL import Image


openai_api_key = os.getenv('OPENAI_API_KEY')

openai_client = OpenAI(api_key=openai_api_key)

MAX_CAPTCHA_ATTEMPTS = 5
OMS_URL = 'https://omsweb.public-safety-cloud.com'
# Yes, "captcha" and "Captcha", as seen in the application traffic
GET_CAPTCHA_CLIENT_URL = f'{OMS_URL}/jtclientweb/captcha/getnewcaptchaclient'
VALIDATE_CAPTCHA_URL = f'{OMS_URL}/jtclientweb/Captcha/validatecaptcha'
# This key (like others) is typo'd. Using a variable here so it's easy to update if they fix the typo.
CAPTCHA_REQUIRED_KEY = 'captchaRequred'

# Decode the base64 image
def extract_text_from_inline_image(inline_image: str) -> str:
    img_prefix = 'data:image/gif;base64,'
    if not inline_image.startswith(img_prefix):
        raise ValueError(f'Unexpected image format: {inline_image}', file=sys.stderr)

    image_base64 = inline_image[len(img_prefix):]
    return extract_text_from_base64(image_base64)

def extract_text_from_base64(base64_image: str) -> str:
    image_data = base64.b64decode(base64_image)
    image = Image.open(BytesIO(image_data))
    return extract_text_from_image(image)

def extract_text_from_image(image: str) -> str:
    """
    Extract text from image, removing any whitespace
    """
    # Pre-process to improve Tesseract performance
    # Convert the image to an OpenCV format
    image_cv = cv2.cvtColor(np.array(image), cv2.COLOR_RGB2BGR)
    # Convert to grayscale
    gray_image = cv2.cvtColor(image_cv, cv2.COLOR_BGR2GRAY)
    # Apply thresholding to get a binary image
    _, binary_image = cv2.threshold(gray_image, 150, 255, cv2.THRESH_BINARY_INV)
    #cv2.imshow('Enhanced', binary_image)
    #cv2.waitKey(0)
    # Convert back to PIL image
    enhanced_image = Image.fromarray(cv2.cvtColor(binary_image, cv2.COLOR_GRAY2RGB))

    text = pytesseract.image_to_string(enhanced_image, config='--psm 8').strip()

    # Remove any whitespace in the output
    return ''.join(text.split())


@dataclass
class Jail:
    """
    TODO docstring
    """
    # Name of the jail, as it appears in the URL
    name: str
    session: requests.Session
    # Captcha key for this jail; updates ... sometimes? But has to be provided per-request.
    captcha_key: str
    #TODO improve typing here
    #TODO consider using our own terminology here; "offenders", really???
    offenders: dict[str, Any]
    # Each request after validation updates this key!
    offender_view_key: str

    @classmethod
    def connect(cls, jail_name: str) -> 'Jail':
        """
        TODO docstring
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
        #TODO handle KeyError here
        try:
            captcha_required = resp_data[CAPTCHA_REQUIRED_KEY]
            jail.captcha_key = resp_data['captchaKey']
            offenders = resp_data['offenders']
            # (each request updates this!)
            jail.offender_view_key = resp_data['offenderViewKey']
        except KeyError as e:
            print(resp_data, file=sys.stderr)
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
        #TODO better return type
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
                #TODO better logging, raise here
                print(e, file=sys.stderr)
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
                #TODO better logging, raise here
                print(resp_data, file=sys.stderr)
                print(data)
                raise e

            if captcha_matched:
                print(f'Matched captcha "{text}" and got captcha key "{captcha_key}"')
                self.captcha_key = captcha_key
                return captcha_key
            print(f'({i}/{MAX_CAPTCHA_ATTEMPTS}) Failed to match captcha. Detected "{text}".')

        #TODO raise on failure, or just return None?
        print('Failed to validate captcha')

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
