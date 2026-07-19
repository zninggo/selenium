# Selenium / WebDriver client for Go (maintained fork)

**Language:** **English** | [简体中文](README.zh-CN.md)

| | |
|---|---|
| **Module** | `github.com/zninggo/selenium` |
| **Upstream** | [tebeka/selenium](https://github.com/tebeka/selenium) (largely unmaintained since ~2021) |
| **Status** | Self-use first; public best-effort |
| **Latest** | `v0.12.0` |
| **Notable delta** | ChromeDriver 115+ / W3C find & SendKeys / Selenium 4 / CDP / Shadow DOM / HTTP reliability / multi-OS drivers / smoke + browser CI |

[![Go Reference](https://pkg.go.dev/badge/github.com/zninggo/selenium.svg)](https://pkg.go.dev/github.com/zninggo/selenium)
[![CI](https://github.com/zninggo/selenium/actions/workflows/ci.yml/badge.svg)](https://github.com/zninggo/selenium/actions/workflows/ci.yml)

This is a [WebDriver][webdriver] client for [Go][go]. It supports the
[WebDriver protocol][webdriver] and works with [ChromeDriver][chromedriver],
[Geckodriver][geckodriver], and [Selenium][selenium] server (3 and 4).

This repository is a maintained fork of [tebeka/selenium](https://github.com/tebeka/selenium).
Upstream commits may be cherry-picked as needed; there is no automatic merge of `upstream/master`.

[selenium]: https://www.selenium.dev/
[webdriver]: https://www.w3.org/TR/webdriver/
[go]: https://go.dev/
[geckodriver]: https://github.com/mozilla/geckodriver
[chromedriver]: https://googlechromelabs.github.io/chrome-for-testing/

## Installing

```text
go get github.com/zninggo/selenium@latest
```

Requires **Go 1.22+** and a working WebDriver stack (browser + driver, or Selenium server).

### Migrating from tebeka/selenium

1. Rewrite imports: `github.com/tebeka/selenium` → `github.com/zninggo/selenium`
2. `go get github.com/zninggo/selenium@v0.12.0`
3. `go mod tidy`

## Quick start (recommended)

### ChromeDriver (direct, ChromeDriver 115+)

ChromeDriver serves sessions at the **root** URL — **do not** append `/wd/hub`.

```go
package main

import (
	"fmt"

	"github.com/zninggo/selenium"
	"github.com/zninggo/selenium/chrome"
)

func main() {
	const (
		chromeDriverPath = "chromedriver" // or path from vendor/init.go
		port             = 9515
	)

	svc, err := selenium.NewChromeDriverService(chromeDriverPath, port)
	if err != nil {
		panic(err)
	}
	defer svc.Stop()

	caps := selenium.Capabilities{"browserName": "chrome"}
	caps.AddChrome(chrome.Capabilities{
		Args: []string{"--headless=new", "--no-sandbox"},
		W3C:  true,
	})

	// Root URL — not http://127.0.0.1:9515/wd/hub
	wd, err := selenium.NewRemote(caps, fmt.Sprintf("http://127.0.0.1:%d", port))
	if err != nil {
		panic(err)
	}
	defer wd.Quit()

	if err := wd.Get("https://golang.org"); err != nil {
		panic(err)
	}
	fmt.Println(wd.Title())
}
```

Also see `ExampleChromeDriver` in [example_test.go](example_test.go).

### Selenium 4 standalone

```go
svc, err := selenium.NewSeleniumServiceV4("selenium-server-4.x.jar", 4444)
// ...
wd, err := selenium.NewRemote(caps, "http://127.0.0.1:4444") // root URL, no /wd/hub
```

See `Example_selenium4` in [example_test.go](example_test.go).

### Selenium 3 (legacy)

`NewSeleniumService` still targets Selenium 3 (`GridLauncherV3`) and listens under **`/wd/hub`**:

```go
svc, err := selenium.NewSeleniumService("selenium-server.jar", 4444)
wd, err := selenium.NewRemote(caps, "http://127.0.0.1:4444/wd/hub")
```

See the original `Example` in [example_test.go](example_test.go).

### URL prefix cheat sheet

| Backend | Helper | `NewRemote` urlPrefix |
|---------|--------|------------------------|
| ChromeDriver 115+ | `NewChromeDriverService` | `http://127.0.0.1:<port>` |
| Geckodriver | `NewGeckoDriverService` | `http://127.0.0.1:<port>` |
| Selenium 4 | `NewSeleniumServiceV4` | `http://127.0.0.1:<port>` |
| Selenium 3 | `NewSeleniumService` | `http://127.0.0.1:<port>/wd/hub` |

## Shadow DOM vs iframe

- **iframe**: separate document — use `SwitchFrame` (already supported).
- **Shadow DOM**: component internal tree on the same page — use `host.GetShadowRoot()` then find inside the returned `ShadowRoot`.

```go
host, err := wd.FindElement(selenium.ByID, "host")
root, err := host.GetShadowRoot()
btn, err := root.FindElement(selenium.ByID, "inner-btn")
btn.Click()
```

Only **open** shadow roots are accessible (W3C / ChromeDriver).

## Behavior notes (this fork)

| Topic | Behavior |
|-------|----------|
| HTTP client | Default `HTTPClient` timeout is **120s** (replace `selenium.HTTPClient` if needed) |
| Request headers | JSON bodies send `Content-Type: application/json` |
| Cookies | Optional fields use `omitempty` (zero `Expiry`/`Path`/… are omitted) |
| W3C find | `ByID` / `ByName` / `ByClassName` map to CSS under W3C mode |
| W3C SendKeys | Sends both `text` and `value` string list |
| Service debug | Selenium 3 `-debug` only when `SetDebug(true)` |
| Linux teardown | Process group + `Pdeathsig` to reduce orphan drivers |
| CDP | `ExecuteCDPCommand(cmd, params)` via ChromeDriver `goog/cdp/execute` |
| Shadow DOM | `elem.GetShadowRoot()` then `root.FindElement` / `FindElements` (open roots) |
| iframe | Already supported via `SwitchFrame` |

## Downloading Dependencies

Primarily for local tests / smoke:

```text
$ cd vendor
$ go run init.go --alsologtostderr --download_browsers --download_latest
$ cd ..
```

Asset selection uses `GOOS`/`GOARCH` (linux64, mac-x64, mac-arm64, win32, win64).
Chrome comes from [Chrome for Testing](https://googlechromelabs.github.io/chrome-for-testing/) Stable.
Firefox on macOS/Windows may download installers/DMGs that need manual install.

## Documentation

- API: https://pkg.go.dev/github.com/zninggo/selenium
- Examples: [example_test.go](example_test.go)
- Main-path smoke: `TestSmokeChrome` in [smoke_test.go](smoke_test.go)

## Smoke test (optional)

Needs Chrome + ChromeDriver. Without them the test **skips** (default CI stays green; runners that already have Chrome may execute it).

```text
cd vendor && go run init.go --alsologtostderr --download_browsers && cd ..
go test -mod=mod -count=1 -timeout=5m -run TestSmokeChrome -v
```

## Known Issues

Many failures come from the browser/driver, not this client. Please
[file an issue](https://github.com/zninggo/selenium/issues/new) if the client misbehaves.

### Selenium 2

No longer supported.

### Selenium 3

1. [Selenium 3 NewSession does not implement the W3C-specified parameters](https://github.com/SeleniumHQ/selenium/issues/2827).

### Geckodriver (Standalone)

1. [Geckodriver does not support the Log API](https://github.com/mozilla/geckodriver/issues/284).
2. [Click issues](https://github.com/mozilla/geckodriver/issues/1007).
3. [Control characters may need a terminating null key](https://github.com/mozilla/geckodriver/issues/665).

### Chromedriver

1. [Headless Chrome does not support running extensions](https://crbug.com/706008).
2. Use the **root** service URL with ChromeDriver 115+ (not `/wd/hub`).

## Changelog (summary)

| Version | Highlights |
|---------|------------|
| v0.10.0 | Fork rehome, Go 1.22, GitHub Actions |
| v0.10.1 | HTTP body close, client timeout, CurrentURL, cookie expiry |
| v0.10.2 | W3C class/name find, `NewSeleniumServiceV4` |
| v0.10.3 | Linux process-group / Pdeathsig orphan cleanup |
| v0.10.4 | Multi-OS `vendor/init.go`, Chrome for Testing |
| v0.10.5 | Content-Type, W3C SendKeys value list, cookie omitempty, gated `-debug` |
| v0.10.6 | `TestSmokeChrome`; ChromeDriver tests without `/wd/hub` |
| v0.10.7 | Smoke read-back via `GetProperty` on headless CI |
| v0.10.8 | README modern usage; `ExampleChromeDriver` / `Example_selenium4` |
| v0.11.0 | `ExecuteCDPCommand`; optional `browser` workflow |
| v0.11.1 | Chinese README (`README.zh-CN.md`) + language switcher |
| v0.12.0 | Shadow DOM: `GetShadowRoot` + find inside open shadow roots |

Full notes: [ChangeLog](ChangeLog) and [Releases](https://github.com/zninggo/selenium/releases).

### Optional browser CI

Manually run (or weekly schedule) the **browser** workflow:
Actions → browser → Run workflow.
It downloads Chrome for Testing and runs `TestSmokeChrome` (including CDP).

### Optional next

1. Feature PRs (Print, Select helpers, remote file upload)

## Breaking Changes (historical)

### 22 August 2017

The `Version` constant was removed as it is unused.

### 18 April 2017

The Log method was changed to accept a typed constant for the type of log to
retrieve, instead of a raw string. The return value was also changed to provide
a more idiomatic type.

## Hacking

Patches are welcome through GitHub pull requests. Please ensure that:

1. A test is added for anything more than a trivial change and that the existing tests pass.
2. `gofmt` has been run on the changed files. Optional pre-commit hook:

```text
ln -s ../../misc/git/pre-commit .git/hooks/pre-commit
```

[Issues](https://github.com/zninggo/selenium/issues)

### Testing Locally

```text
sudo apt-get install xvfb openjdk-11-jre   # if needed
go test -mod=mod ./...
```

Top-level browser tests skip when binaries are missing:

- `TestChrome` / `TestSmokeChrome`
- `TestFirefoxSelenium3` / `TestFirefoxGeckoDriver`
- `TestHTMLUnit` (needs Java + JARs)

Configure paths with `go test` flags (see `go test -args -help`).

### Testing With Docker

```text
go test --docker
```

Or:

```text
docker build -t go-selenium testing/
docker run --volume=$(pwd):/code --workdir=/code -it go-selenium bash
# inside: testing/docker-test.sh
```

### Testing With Sauce Labs

```text
go test --test.run=TestSauce --test.timeout=20m \
  --experimental_enable_sauce \
  --sauce_user_name=[username] \
  --sauce_access_key=[access key]
```

## License

MIT — see [LICENSE](LICENSE).
