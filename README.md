# JTT - a JailTracker tracker

JTT is a project for monitoring carceral facilities who publish their roster via JailTracker.

In particular, JTT fetches data via the JailTracker API for the purpose of gathering data for the sake of police
oversight. For example - how long are individuals held without charges? In what jails?

JTT is expressly *not* intended for obtaining personal information about incarcerated individuals (who cannot consent,)
obtaining mugshots, etc.

## Usage
Currently, I'm not publishing binaries for the project. You need:

* A working Go installation
* An OpenAI API key set in the `JTT_OPENAI_API_KEY` environment variable.
    * I store this in `.env`: `export JTT_OPENAI_API_KEY='ASDF'`
    * If you can roll a better text extraction model that we can run locally, please let me know!

You can configure which jails to monitor and where to store data in `config.json`. For example, production data might be better stored in `/var/lib/jtt`, but the default is `./cache` for local development.

To run: `. .env && go run .`

## About JTT

JTT was initially a collaboration with another project:
@bfeldman89's [jail_scrapers](https://github.com/bfeldman89/jail_scrapers),
which focuses on gathering data about Mississippi jails.
Please see https://bfeldman89.com/projects/jails/ for information about this work.

As JailTracker is utilized in many places outside of Mississippi,
this code has been separated from `jail_scrapers` for those interested in similar civic hacking projects.

JTT uses the API of the JailTracker software suite
in order to aggregate information about inmates for civic purposes.
We're interested in answering questions about treatment of inmates, including but not limited to:

* How long are inmates held pre-trial? Pre-arraignment?
* How long are inmates being held without being charged at all, or when charges are dropped before trial?
* Does this data vary along demographic or geographic lines?

This project was on hiatus for a few years due to JailTracker implementing a captcha system, but that's currently not an issue.
While it does bypass captchas, JTT does default to a VERY generous rate limit to avoid impacting response times for other users.

When re-writing the project to account for the updated JailTracker API, I also ported the code from Python to Go.
This is a matter of personal taste, but `jailtracker.py` does provide a proof-of-concept module for interacting
with the current API.

## Known issues with available data

* Some jails (e.g. Harrison County, MS) require that you provide at least two characters in the "Last Name" field, otherwise no data will be provided.
* Some jails don't consistently provide case or charge data.
* Jails do not consistently use all the features of JailTracker. For example, there are some redundant date/time fields in the API, but often only one will be set.
* Most data seems to be input by hand, and is often quite messy. I'll try to document this on the repo's wiki at some point.
