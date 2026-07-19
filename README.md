# Selenium / WebDriver client for Go (maintained fork)

| | |
|---|---|
| **Module** | `github.com/zninggo/selenium` |
| **Upstream** | [tebeka/selenium](https://github.com/tebeka/selenium) (largely unmaintained since ~2021) |
| **Status** | Self-use first; public best-effort |
| **Notable delta** | ChromeDriver 115+ session / URL-base compatibility |

[![Go Reference](https://pkg.go.dev/badge/github.com/zninggo/selenium.svg)](https://pkg.go.dev/github.com/zninggo/selenium)
[![CI](https://github.com/zninggo/selenium/actions/workflows/ci.yml/badge.svg)](https://github.com/zninggo/selenium/actions/workflows/ci.yml)

This is a [WebDriver][webdriver] client for [Go][go]. It supports the
[WebDriver protocol][webdriver] and has been tested with various versions of
[Selenium WebDriver][selenium], Firefox and [Geckodriver][geckodriver], and
Chrome and [ChromeDriver][chromedriver].

This repository is a maintained fork of [tebeka/selenium](https://github.com/tebeka/selenium).
Upstream commits may be cherry-picked as needed; there is no automatic merge of `upstream/master`.

[selenium]: http://seleniumhq.org/
[webdriver]: https://www.w3.org/TR/webdriver/
[go]: http://golang.org/
[geckodriver]: https://github.com/mozilla/geckodriver
[chromedriver]: https://sites.google.com/a/chromium.org/chromedriver/

## Installing

```text
go get github.com/zninggo/selenium@latest
```

Requires Go 1.22+ and a working WebDriver stack (browser + driver, or Selenium server).

### Migrating from tebeka/selenium

1. Rewrite imports: `github.com/tebeka/selenium` → `github.com/zninggo/selenium`
2. `go get github.com/zninggo/selenium@v0.10.0`
3. `go mod tidy`

### Downloading Dependencies

We provide a means to download the ChromeDriver binary, the Firefox binary, the
Selenium WebDriver JARs, and the Sauce Connect proxy binary. This is primarily
intended for testing.

```text
$ cd vendor
$ go run init.go --alsologtostderr --download_browsers --download_latest
$ cd ..
```

Re-run this periodically to get up-to-date versions of these binaries.
Note: the helper currently targets Linux asset URLs.

## Documentation

API docs: https://pkg.go.dev/github.com/zninggo/selenium

See [example_test.go](https://github.com/zninggo/selenium/blob/master/example_test.go)
and unit tests for usage.

## Known Issues

Any issues are usually because the underlying browser automation framework has a
bug or inconsistency. Where possible, we try to cover up these underlying
problems in the client, but sometimes workarounds require higher-level
intervention.

Please feel free to [file an issue][issue] if this client doesn't work as
expected.

[issue]: https://github.com/zninggo/selenium/issues/new

Below are known issues that affect the usage of this API. There are likely
others filed on the respective issue trackers.

### Selenium 2

No longer supported.

### Selenium 3

1.  [Selenium 3 NewSession does not implement the W3C-specified parameters](https://github.com/SeleniumHQ/selenium/issues/2827).

### Geckodriver (Standalone)

1.  [Geckodriver does not support the Log API](https://github.com/mozilla/geckodriver/issues/284)
    because it
    [hasn't been defined in the spec yet](https://github.com/w3c/webdriver/issues/406).
2.  Firefox via Geckodriver (and also through Selenium)
    [doesn't handle clicking on an element](https://github.com/mozilla/geckodriver/issues/1007).
3.  Firefox via Geckodriver doesn't handle sending control characters
    [without appending a terminating null key](https://github.com/mozilla/geckodriver/issues/665).

### Chromedriver

1. [Headless Chrome does not support running extensions](https://crbug.com/706008).

## Backlog

Done:
- v0.10.1: response body Close, HTTPClient timeout, service shutdown Kill fallback, CurrentURL nil-safe, Cookie.Expiry omitempty
- v0.10.2: `ByClassName`/`ByName` W3C locator mapping, `NewSeleniumServiceV4`

Still open (self-use priority):

1. Broader process-group / Pdeathsig orphan cleanup
2. Multi-OS modern binary download in `vendor/init.go`
3. Optional browser integration CI

## Breaking Changes

There are a number of upcoming changes that break backward compatibility in an
effort to improve and adapt the existing API. They are listed here:

### 22 August 2017

The `Version` constant was removed as it is unused.

### 18 April 2017

The Log method was changed to accept a typed constant for the type of log to
retrieve, instead of a raw string. The return value was also changed to provide
a more idiomatic type.

## Hacking

Patches are welcome through GitHub pull requests. Please ensure that:

1.  A test is added for anything more than a trivial change and that the
    existing tests pass. See below for instructions on setting up your test
    environment.
2.  Please ensure that `gofmt` has been run on the changed files before
    committing. Install a pre-commit hook with the following command:

    $ ln -s ../../misc/git/pre-commit .git/hooks/pre-commit

See [the issue tracker][issues] for features that need implementing.

[issues]: https://github.com/zninggo/selenium/issues

### Testing Locally

Install `xvfb` and Java if they is not already installed, e.g.:

    sudo apt-get install xvfb openjdk-11-jre

Run the tests:

    $ go test

*   There is one top-level test for each of:

    1.  Chromium and ChromeDriver.
    2.  A new version of Firefox and Selenium 3.
    3.  HTMLUnit, a Java-based lightweight headless browser implementation.
    4.  A new version of Firefox directly against Geckodriver.

    There are subtests that are shared between both top-level tests.

*   To run only one of the top-level tests, pass one of:

    *   `-test.run=TestFirefoxSelenium3`,
    *   `-test.run=TestFirefoxGeckoDriver`,
    *   `-test.run=TestHTMLUnit`, or
    *   `-test.run=TestChrome`.

    To run a specific subtest, pass `-test.run=Test<Browser>/<subtest>` as
    appropriate. This flag supports regular expressions.

*   If the Chrome or Firefox binaries, the Selenium JAR, the Geckodriver binary,
    or the ChromeDriver binary cannot be found, the corresponding tests will be
    skipped.

*   The binaries and JAR under test can be configured by passing flags to `go
    test`. See the available flags with `go test --arg --help`.

*   Add the argument `-test.v` to see detailed output from the test automation
    framework.

### Testing With Docker

To ensure hermeticity, we also have tests that run under Docker. You will need
an installed and running Docker system.

To run the tests under Docker, run:

    $ go test --docker

This will create a new Docker container and run the tests in it. (Note: flags
supplied to this invocation are not curried through to the `go test` invocation
within the Docker container).

For debugging Docker directly, run the following commands:

    $ docker build -t go-selenium testing/
    $ docker run --volume=$(pwd):/code --workdir=/code -it go-selenium bash
    root@6c7951e41db6:/code# testing/docker-test.sh
    ... lots of testing output ...

### Testing With Sauce Labs

Tests can be run using a browser located in the cloud via Sauce Labs.

To run the tests under Sauce, run:

    $ go test --test.run=TestSauce --test.timeout=20m \
      --experimental_enable_sauce \
      --sauce_user_name=[username goes here] \
      --sauce_access_key=[access key goes here]

The Sauce access key can be obtained via
[the Sauce Labs user settings page](https://saucelabs.com/beta/user-settings).

Test results can be viewed through the
[Sauce Labs Dashboard](https://saucelabs.com/beta/dashboard/tests).

## License

This project is licensed under the [MIT][mit] license.

[mit]: https://raw.githubusercontent.com/zninggo/selenium/master/LICENSE
