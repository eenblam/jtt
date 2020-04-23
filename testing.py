import json
import os
from airtable import Airtable

airtab_intakes = Airtable(base_key='app8pZM1MUG46TpOh',
                          table_name='intakes',
                          api_key=os.environ['AIRTABLE_API_KEY'])

airtab_cases = Airtable(base_key='app8pZM1MUG46TpOh',
                        table_name='cases',
                        api_key=os.environ['AIRTABLE_API_KEY'])

airtab_charges = Airtable(base_key='app8pZM1MUG46TpOh',
                          table_name='charges',
                          api_key=os.environ['AIRTABLE_API_KEY'])


with open('results.json', 'r') as json_reader:
    results = json.load(json_reader)

def main():
    for key, data in results.items():
        print(key)
        for result in data:
            result['inmate'].update(result['summary'])
            intake = airtab_intakes.insert(result['inmate'], typecast=True)
            intake_id = {}
            intake_id['intake'] = []
            intake_id['intake'].append(intake['id'])
            for case in result['cases']:
                case.update(intake_id)
                airtab_cases.insert(case, typecast=True)
            for charge in result['charges']:
                charge.update(intake_id)
                airtab_charges.insert(charge, typecast=True)


if __name__ == "__main__":
    main()
