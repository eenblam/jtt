#!/usr/bin/env python3

"""
It's a pain to scrape Google search results due to reCaptcha,
but it's easy to open 20 or so tabs and Ctrl+S each page.

So I searched inurl:omsweb.public-safety-cloud.com/
I saved as 1.html, 2.html, ...

This script just processes each saved page to extract the search results,
which I then copied into Google Sheets for further processing.

(This could be easily modified to just pull general search results.)
"""


import csv
import glob
import os

import bs4

# Just create anything in your cache directory, which is .gitignored
# (You do need to create a ./cache directory)
d = 'cache/search/'
outfile = os.path.join(d, 'results.csv')
output = []

for filepath in sorted(glob.glob(os.path.join(d, '*.html'))):
    print(f'Reading {filepath}')
    with open(filepath, 'r', encoding='utf-8') as f:
        data = f.read()

    soup = bs4.BeautifulSoup(data)
    results = soup.select('a:has(> h3)')
    print(f'Found {len(results)} results in {filepath}')
    for r in results:
        title = r.find('h3').text
        original_url = r['href']

        # jtclientwebofficial goes to old/busted version of JT
        # shorten to jtclientweb if found in URL
        url = original_url.replace('jtclientwebofficial', 'jtclientweb')

        if not url.startswith('https://omsweb.public-safety-cloud.com/'):
            print(f'Skipping {url}')
            continue

        output.append({
            'Title': title,
            'Original URL': original_url,  # For posterity
            'JailTracker URL': url,
            })


print(f'Found {len(output)} total results')

with open(outfile, mode='w', newline='', encoding='utf-8') as file:
    writer = csv.DictWriter(file, fieldnames=('Title', 'JailTracker URL', 'Original URL'))
    writer.writeheader()
    writer.writerows(output)

print(f'Wrote results to {outfile}')
