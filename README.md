# JTT - a JailTracker tracker

JTT is a collaboration with another project:
@bfeldman89's [jail_scrapers](https://github.com/bfeldman89/jail_scrapers),
which focuses on gathering data about Mississippi jails.
Please see https://bfeldman89.com/projects/jails/ for information about this work.

As JailTracker is utilized in many places outside of Mississippi,
this code has been separated from `jail_scrapers` for those interested in similar civic hacking projects.

## About JTT
JTT uses the API of the JailTracker software suite
in order to aggregate information about inmates for civic purposes.
We're interested in answering questions about treatment of inmates, including but not limited to:
* How long are inmates held pre-trial? Pre-arraignment?
* How long are inmates being held without being charged at all, or when charges are dropped before trial?
* Does this data vary along demographic or geographic lines?

JTT is **not** for your mugshot website, and support for downloading mugshots will not be added.

## Usage
See `examples/to_json.py` for a complete example of how to use JTT to get a JSON data set
of however many jails you're interested in.

You can also see `examples/airtable_worker.py` for an example of how to use Airtable as a storage backend.

### JailTracker URLs
When you visit a JailTracker page, have a look at the address bar in your browser.
The URL will look something like "https://omsweb.public-safety-cloud.com/jtclientweb/(S(3r01ougmulvelofaoxij53r4))/jailtracker/index/HANCOCK_COUNTY_MS".
The `(S(3r01ougmulvelofaoxij53r4))` is JailTracker's funky way of handling session keys.
If you try to reuse the above link, you'll probably get a new session key.

However, to get a key without having one in the first place, you could have just gone to
https://omsweb.public-safety-cloud.com/jtclientweb/jailtracker/index/HANCOCK_COUNTY_MS.
Note - this just omits the key component altogether!
JailTracker will respond to your request with `302 Found`,
and you'll be redirected to the address specified in the `Location` header of the response.
This address contains your new session key.

Fortunately, JTT handles that part for you.
The simplest use case looks like this:
```python
import json

hancock = jailtracker.Jail('https://omsweb.public-safety-cloud.com/jtclientweb/jailtracker/index/HANCOCK_COUNTY_MS')
data, err = hancock.process_inmates()
if err is not None:
    print(err)
else:
    print(json.dumps(data, ensure_ascii=False, indent=4))
```

### Finding your jail's URL
Often, a sheriff's department will embed the JailTracker page in their website.
For example, see http://www.hancockso.com/InmateRoster/hancock_inmatelist.html.
(Sorry, no TLS.)

Here's how to get the URL described in the previous section.

Using the Firefox browser:
* Right click inside the JailTracker frame
* Choose "This Frame", then "Show Only This Frame"
* You'll be redirected to the omsweb page. Copy the URL from your address bar.
* Finally, reference the previous section to learn how to remove the unneeded session key from the URL.

Chrome doesn't provide the same feature. Instead, you can:
* Right click inside the JailTracker frame
* Choose "Show frame source"
* The source code for the JailTracker page will open in a new tab. The address bar of this new tab should contain the URL you want. Just strip off the `view-source:` bit at the beginning.
* Finally, reference the previous section to learn how to remove the unneeded session key from the URL.

## Data modeling
To our knowledge, JailTracker doesn't publish its internal data model.

In general, jails do tend to publish the following:
* A **summary** for each inmate (returned as a list for `GetInmates`)
* A detailed **inmate** report for a given arrest number (`GetInmate`)
* A list of **cases** for a given arrest number (`GetCases`)
* A list of **charges** for a given arrest number (`GetCharges`)
    * For some reason, this is done via a POST request, not a GET. Go figure.

Note: some jails are rather inconsistent with what they provide.
For example, only a small percentage of inmates (or none at all)
might have case or charge data available.

### Relational data stores
It's easy to use JTT to produce a JSON file of all the data you pull.
See the included `examples/to_json.py`.

However, if you want to use a relational data store,
you need to know what tables and columns belong in your data model.
To help with this, the script `getcolumns.py` is included.
It reads jail data from a file called `results.json` - which you can produce using `example.py`,
and it produces two files - `column_totals.json` and `column_results.json`.

You can use the two resulting data sets to figure out the following:
* What columns are needed for each data type (table)?
    * `column_totals.json` is the easy way to read this
* For a column of interest, which jails provide that information?
    * See `column_results.json`

### Known issues with available data
Some jails (e.g. Harrison County, MS) require that you provide at least two characters in the "Last Name" field, otherwise no data will be provided.

As previously mentioned, some jails don't consistently provide case or charge data.
