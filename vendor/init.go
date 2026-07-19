// Binary init downloads the necessary files to perform an integration test
// between this WebDriver client and multiple versions of Selenium and
// browsers.
//
// Asset selection is based on GOOS/GOARCH. Unsupported combinations fail with
// a clear error instead of silently downloading Linux-only URLs.
package main

import (
	"context"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"hash"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/google/go-github/v27/github"
)

const (
	// desiredFirefoxVersion is used when --download_latest is false.
	// Update periodically.
	desiredFirefoxVersion = "128.0.3"

	// desiredSelenium3JAR is the Selenium 3 standalone server used by legacy tests.
	desiredSelenium3JAR = "https://github.com/SeleniumHQ/selenium/releases/download/selenium-3.141.59/selenium-server-standalone-3.141.59.jar"

	// chromeForTestingJSON is the Chrome for Testing last-known-good versions API.
	chromeForTestingJSON = "https://googlechromelabs.github.io/chrome-for-testing/last-known-good-versions-with-downloads.json"
)

var (
	downloadBrowsers = flag.Bool("download_browsers", true, "If true, download the Firefox and Chrome browsers.")
	downloadLatest   = flag.Bool("download_latest", false, "If true, download the latest versions where available.")
	httpClient       = &http.Client{Timeout: 5 * time.Minute}
)

type file struct {
	url      string
	name     string
	hash     string
	hashType string // default is sha256
	rename   []string
	browser  bool
}

var files []file

func main() {
	flag.Parse()
	ctx := context.Background()

	platform, err := detectPlatform()
	if err != nil {
		glog.Exitf("%v", err)
	}
	glog.Infof("Downloading assets for %s/%s (platform key %q)", runtime.GOOS, runtime.GOARCH, platform)

	// Selenium 3 standalone (legacy integration tests).
	files = append(files, file{
		url:  desiredSelenium3JAR,
		name: "selenium-server.jar",
	})

	if err := addSauceConnect(platform); err != nil {
		glog.Errorf("Sauce Connect: %v", err)
	}

	if *downloadBrowsers {
		if err := addChromeForTesting(platform, *downloadLatest); err != nil {
			glog.Errorf("Chrome for Testing: %v", err)
		}
		if err := addFirefox(platform, desiredFirefoxVersion, *downloadLatest); err != nil {
			glog.Errorf("Firefox: %v", err)
		}
	}

	if err := addLatestGithubRelease(ctx, "SeleniumHQ", "htmlunit-driver", "htmlunit3?-driver-.*-jar-with-dependencies\\.jar", "htmlunit-driver.jar"); err != nil {
		glog.Errorf("HTMLUnit Driver: %v", err)
	}

	geckoAsset, err := geckodriverAsset(platform)
	if err != nil {
		glog.Errorf("Geckodriver: %v", err)
	} else if err := addLatestGithubRelease(ctx, "mozilla", "geckodriver", geckoAsset, geckodriverLocalName(platform)); err != nil {
		glog.Errorf("Geckodriver: %v", err)
	}

	if *downloadLatest {
		if err := addLatestGithubRelease(ctx, "SeleniumHQ", "selenium", `selenium-server-\d+\.\d+\.\d+\.jar`, "selenium-server-4.jar"); err != nil {
			glog.Errorf("Selenium 4 server: %v", err)
		}
	}

	var wg sync.WaitGroup
	for _, f := range files {
		wg.Add(1)
		f := f
		go func() {
			defer wg.Done()
			if err := handleFile(f); err != nil {
				glog.Exitf("Error handling %s: %s", f.name, err)
			}
		}()
	}
	wg.Wait()
}

func detectPlatform() (string, error) {
	switch runtime.GOOS {
	case "linux":
		if runtime.GOARCH == "amd64" {
			return "linux64", nil
		}
	case "darwin":
		switch runtime.GOARCH {
		case "amd64":
			return "mac-x64", nil
		case "arm64":
			return "mac-arm64", nil
		}
	case "windows":
		switch runtime.GOARCH {
		case "amd64":
			return "win64", nil
		case "386":
			return "win32", nil
		}
	}
	return "", fmt.Errorf("unsupported GOOS/GOARCH %s/%s for browser asset downloads", runtime.GOOS, runtime.GOARCH)
}

func addChromeForTesting(platform string, latest bool) error {
	// Always use last-known-good Stable; "latest" still means Stable tip (not Canary).
	_ = latest
	resp, err := httpClient.Get(chromeForTestingJSON)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("chrome-for-testing JSON HTTP %s", resp.Status)
	}
	var payload struct {
		Channels map[string]struct {
			Version   string `json:"version"`
			Downloads struct {
				Chrome       []struct{ Platform, URL string } `json:"chrome"`
				Chromedriver []struct{ Platform, URL string } `json:"chromedriver"`
			} `json:"downloads"`
		} `json:"channels"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return err
	}
	stable, ok := payload.Channels["Stable"]
	if !ok {
		return fmt.Errorf("Stable channel missing from chrome-for-testing JSON")
	}
	chromeURL, err := findPlatformURL(stable.Downloads.Chrome, platform)
	if err != nil {
		return fmt.Errorf("chrome: %w", err)
	}
	driverURL, err := findPlatformURL(stable.Downloads.Chromedriver, platform)
	if err != nil {
		return fmt.Errorf("chromedriver: %w", err)
	}
	glog.Infof("Chrome for Testing Stable %s (%s)", stable.Version, platform)

	chromeArchive := "chrome-" + platform + ".zip"
	driverArchive := "chromedriver-" + platform + ".zip"
	files = append(files,
		file{
			name:    chromeArchive,
			url:     chromeURL,
			browser: true,
			rename:  chromeRename(platform),
		},
		file{
			name:   driverArchive,
			url:    driverURL,
			rename: chromedriverRename(platform),
		},
	)
	return nil
}

func findPlatformURL(items []struct{ Platform, URL string }, platform string) (string, error) {
	for _, it := range items {
		if it.Platform == platform {
			return it.URL, nil
		}
	}
	return "", fmt.Errorf("no download for platform %q", platform)
}

func chromeRename(platform string) []string {
	// Keep linux path compatible with selenium_test.go default vendor/chrome-linux/chrome.
	switch platform {
	case "linux64":
		return []string{"chrome-linux64", "chrome-linux"}
	case "mac-x64":
		return []string{"chrome-mac-x64", "chrome-mac"}
	case "mac-arm64":
		return []string{"chrome-mac-arm64", "chrome-mac"}
	case "win64":
		return []string{"chrome-win64", "chrome-win"}
	case "win32":
		return []string{"chrome-win32", "chrome-win"}
	default:
		return nil
	}
}

func chromedriverRename(platform string) []string {
	switch platform {
	case "linux64":
		return []string{"chromedriver-linux64/chromedriver", "chromedriver"}
	case "mac-x64":
		return []string{"chromedriver-mac-x64/chromedriver", "chromedriver"}
	case "mac-arm64":
		return []string{"chromedriver-mac-arm64/chromedriver", "chromedriver"}
	case "win64":
		return []string{"chromedriver-win64/chromedriver.exe", "chromedriver.exe"}
	case "win32":
		return []string{"chromedriver-win32/chromedriver.exe", "chromedriver.exe"}
	default:
		return nil
	}
}

func addFirefox(platform, version string, latest bool) error {
	// Browser packages: Linux is tar.bz2; Windows zip; macOS dmg is not auto-extracted here.
	switch {
	case latest && (platform == "linux64"):
		files = append(files, file{
			url:     "https://download.mozilla.org/?product=firefox-latest-ssl&os=linux64&lang=en-US",
			name:    "firefox.tar.bz2",
			browser: true,
		})
		return nil
	case platform == "linux64":
		files = append(files, file{
			url: "https://download-installer.cdn.mozilla.net/pub/firefox/releases/" +
				url.PathEscape(version) + "/linux-x86_64/en-US/firefox-" + url.PathEscape(version) + ".tar.bz2",
			name:    "firefox.tar.bz2",
			browser: true,
		})
		return nil
	case platform == "win64":
		v := version
		if latest {
			// Product link resolves to latest; store as zip name for extraction.
			files = append(files, file{
				url:     "https://download.mozilla.org/?product=firefox-latest-ssl&os=win64&lang=en-US",
				name:    "firefox-win64.exe",
				browser: true,
			})
			glog.Warningf("Firefox Windows download is an installer (%s); extract/install manually if needed", v)
			return nil
		}
		files = append(files, file{
			url: "https://download-installer.cdn.mozilla.net/pub/firefox/releases/" +
				url.PathEscape(version) + "/win64/en-US/Firefox%20Setup%20" + url.PathEscape(version) + ".exe",
			name:    "firefox-win64.exe",
			browser: true,
		})
		glog.Warningf("Firefox Windows asset is an installer; not auto-extracted")
		return nil
	case strings.HasPrefix(platform, "mac-"):
		osLabel := "osx"
		if latest {
			files = append(files, file{
				url:     "https://download.mozilla.org/?product=firefox-latest-ssl&os=" + osLabel + "&lang=en-US",
				name:    "firefox.dmg",
				browser: true,
			})
		} else {
			files = append(files, file{
				url: "https://download-installer.cdn.mozilla.net/pub/firefox/releases/" +
					url.PathEscape(version) + "/mac/en-US/Firefox%20" + url.PathEscape(version) + ".dmg",
				name:    "firefox.dmg",
				browser: true,
			})
		}
		glog.Warningf("Firefox macOS asset is a DMG; mount/extract manually if needed")
		return nil
	default:
		return fmt.Errorf("firefox browser download not configured for %s", platform)
	}
}

func geckodriverAsset(platform string) (string, error) {
	switch platform {
	case "linux64":
		return `geckodriver-.*-linux64\.tar\.gz`, nil
	case "mac-x64":
		return `geckodriver-.*-macos\.tar\.gz`, nil
	case "mac-arm64":
		return `geckodriver-.*-macos-aarch64\.tar\.gz`, nil
	case "win64":
		return `geckodriver-.*-win64\.zip`, nil
	case "win32":
		return `geckodriver-.*-win32\.zip`, nil
	default:
		return "", fmt.Errorf("no geckodriver asset pattern for %s", platform)
	}
}

func geckodriverLocalName(platform string) string {
	if strings.HasPrefix(platform, "win") {
		return "geckodriver.zip"
	}
	return "geckodriver.tar.gz"
}

func addSauceConnect(platform string) error {
	// Sauce Connect 4.9.2 multi-OS packages.
	const ver = "4.9.2"
	switch platform {
	case "linux64":
		files = append(files, file{
			url:    fmt.Sprintf("https://saucelabs.com/downloads/sc-%s-linux.tar.gz", ver),
			name:   "sauce-connect.tar.gz",
			rename: []string{fmt.Sprintf("sc-%s-linux", ver), "sauce-connect"},
		})
	case "mac-x64", "mac-arm64":
		files = append(files, file{
			url:    fmt.Sprintf("https://saucelabs.com/downloads/sc-%s-osx.zip", ver),
			name:   "sauce-connect.zip",
			rename: []string{fmt.Sprintf("sc-%s-osx", ver), "sauce-connect"},
		})
	case "win32", "win64":
		files = append(files, file{
			url:    fmt.Sprintf("https://saucelabs.com/downloads/sc-%s-win32.zip", ver),
			name:   "sauce-connect.zip",
			rename: []string{fmt.Sprintf("sc-%s-win32", ver), "sauce-connect"},
		})
	default:
		return fmt.Errorf("no Sauce Connect package for %s", platform)
	}
	return nil
}

// addLatestGithubRelease adds a file to the list of files to download from the
// latest release of the specified Github repository that matches the asset
// name. The file will be downloaded to localFileName.
func addLatestGithubRelease(ctx context.Context, owner, repo, assetName, localFileName string) error {
	client := github.NewClient(httpClient)

	rel, _, err := client.Repositories.GetLatestRelease(ctx, owner, repo)
	if err != nil {
		return err
	}
	assetNameRE, err := regexp.Compile(assetName)
	if err != nil {
		return fmt.Errorf("invalid asset name regular expression %q: %s", assetName, err)
	}
	for _, a := range rel.Assets {
		if !assetNameRE.MatchString(a.GetName()) {
			continue
		}
		u := a.GetBrowserDownloadURL()
		if u == "" {
			return fmt.Errorf("%s does not have a download URL", a.GetName())
		}
		files = append(files, file{
			name: localFileName,
			url:  u,
		})
		return nil
	}

	return fmt.Errorf("release asset %s not found at https://github.com/%s/%s/releases", assetName, owner, repo)
}

func handleFile(file file) error {
	if file.browser && !*downloadBrowsers {
		glog.Infof("Skipping %q because --download_browsers is not set.", file.name)
		return nil
	}
	if file.hash != "" && fileSameHash(file) {
		glog.Infof("Skipping file %q which has already been downloaded.", file.name)
	} else {
		glog.Infof("Downloading %q from %q", file.name, file.url)
		if err := downloadFile(file); err != nil {
			return err
		}
	}

	switch {
	case strings.HasSuffix(file.name, ".zip"):
		glog.Infof("Unzipping %q", file.name)
		if err := exec.Command("unzip", "-o", file.name).Run(); err != nil {
			return fmt.Errorf("error unzipping %q: %v", file.name, err)
		}
	case strings.HasSuffix(file.name, ".tar.gz") || strings.HasSuffix(file.name, ".tgz"):
		glog.Infof("Extracting %q", file.name)
		if err := exec.Command("tar", "-xzf", file.name).Run(); err != nil {
			return fmt.Errorf("error extracting %q: %v", file.name, err)
		}
	case strings.HasSuffix(file.name, ".bz2"):
		glog.Infof("Extracting %q", file.name)
		if err := exec.Command("tar", "-xjf", file.name).Run(); err != nil {
			return fmt.Errorf("error extracting %q: %v", file.name, err)
		}
	case strings.HasSuffix(file.name, ".dmg"), strings.HasSuffix(file.name, ".exe"):
		glog.Infof("Skipping auto-extract for %q (install manually if required)", file.name)
	}

	if rename := file.rename; len(rename) == 2 {
		glog.Infof("Renaming %q to %q", rename[0], rename[1])
		os.RemoveAll(rename[1]) // Ignore error.
		// If source is nested, ensure parent path exists for destination.
		if err := os.Rename(rename[0], rename[1]); err != nil {
			// Fall back to moving a single file when directory layout differs slightly.
			matches, _ := filepath.Glob(rename[0])
			if len(matches) == 1 {
				if err2 := os.Rename(matches[0], rename[1]); err2 != nil {
					glog.Warningf("Error renaming %q to %q: %v", rename[0], rename[1], err)
				}
			} else {
				glog.Warningf("Error renaming %q to %q: %v", rename[0], rename[1], err)
			}
		}
	}
	return nil
}

func downloadFile(file file) (err error) {
	f, err := os.Create(file.name)
	if err != nil {
		return fmt.Errorf("error creating %q: %v", file.name, err)
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("error closing %q: %v", file.name, closeErr)
		}
	}()

	resp, err := httpClient.Get(file.url)
	if err != nil {
		return fmt.Errorf("%s: error downloading %q: %v", file.name, file.url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: download %q: HTTP %s", file.name, file.url, resp.Status)
	}
	if file.hash != "" {
		var h hash.Hash
		switch strings.ToLower(file.hashType) {
		case "md5":
			h = md5.New()
		case "sha1":
			h = sha1.New()
		default:
			h = sha256.New()
		}
		if _, err := io.Copy(io.MultiWriter(f, h), resp.Body); err != nil {
			return fmt.Errorf("%s: error downloading %q: %v", file.name, file.url, err)
		}
		if h := hex.EncodeToString(h.Sum(nil)); h != file.hash {
			return fmt.Errorf("%s: got %s hash %q, want %q", file.name, file.hashType, h, file.hash)
		}
	} else {
		if _, err := io.Copy(f, resp.Body); err != nil {
			return fmt.Errorf("%s: error downloading %q: %v", file.name, file.url, err)
		}
	}
	return nil
}

func fileSameHash(file file) bool {
	if _, err := os.Stat(file.name); err != nil {
		return false
	}
	var h hash.Hash
	switch strings.ToLower(file.hashType) {
	case "md5":
		h = md5.New()
	default:
		h = sha256.New()
	}
	f, err := os.Open(file.name)
	if err != nil {
		return false
	}
	defer f.Close()

	if _, err := io.Copy(h, f); err != nil {
		return false
	}

	sum := hex.EncodeToString(h.Sum(nil))
	if sum != file.hash {
		glog.Warningf("File %q: got hash %q, expect hash %q", file.name, sum, file.hash)
		return false
	}
	return true
}
