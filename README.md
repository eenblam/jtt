# JTT - a JailTracker tracker

JTT is a project for monitoring carceral facilities who publish their roster via [JailTracker](https://jailtracker.com/).

In particular, JTT fetches data via the JailTracker API for research into treatment of people who are incarcerated.
For example, how long are people held without charges? In what jails?

JTT is expressly *not* intended for obtaining personal information about incarcerated individuals, obtaining mugshots, etc.

## Usage
Currently, I'm not publishing binaries for the project. You need:

* A compiler for the Go programming language, which you can install [here](https://go.dev/doc/install)
* An [OpenAI API key](https://platform.openai.com/docs/quickstart) set in the `JTT_OPENAI_API_KEY` environment variable.
    * I store this in `./.env`: `export JTT_OPENAI_API_KEY='ASDF'`
    * This service is used for detecting text in images. If you have ideas for a comparable text extraction model that can be run locally, please let me know!

You can configure which jails to monitor and where to store data in `config.json`. For example, production data might be better stored in `/var/lib/jtt`, but the default is `./cache` for local development.

To run: `. .env && go run .`

## About JTT
See [the wiki](https://github.com/eenblam/jtt/wiki) to learn more about the project, how the JailTracker API works, problems encountered with data from JailTracker, how to find jails, etc.

JTT is now written in the Go programming language, but was originally a Python project. It still includes a proof-of-concept script (`jailtracker.py`) for anyone wanting to work with the current Jailtracker API in Python.