#!/usr/bin/env python3

"""
Convert CSV rows to list of JSON objects. This is helpful for fleshing out config.json with more jails.

Here's how I'm using this:

* I run Google searches for jails as described in google.py
* import the result into a spreadsheet
* manually review and code the data (does the URL work? Does the jail actually return data?)
* save a CSV
* run this script to get a JSON list of jails
* incorporate the list of jails into the existing config.json
"""

import csv
import json

csv_filepath = 'more_jails.csv'
json_filepath = 'more_jails.json'

# Read the CSV file and convert it to a list of dictionaries
with open(csv_filepath, mode='r', newline='', encoding='utf-8') as csv_file:
	csv_reader = csv.DictReader(csv_file)
	data = [row for row in csv_reader]

# Write the list of dictionaries to a JSON file
with open(json_filepath, mode='w', encoding='utf-8') as json_file:
	json.dump(data, json_file, indent=2)
