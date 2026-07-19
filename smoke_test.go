package selenium_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/zninggo/selenium"
	"github.com/zninggo/selenium/chrome"
)

// TestSmokeChrome exercises the main paths fixed in this fork against a real
// ChromeDriver + Chrome when available. It skips cleanly when binaries are
// missing so default CI stays green without browsers.
//
// Covers:
//   - ChromeDriver session creation (no /wd/hub, W3C capabilities)
//   - FindElement ByID / ByClassName (W3C locator mapping)
//   - SendKeys with W3C value list
//   - AddCookie / GetCookies (optional fields omitempty)
//   - CurrentURL nil-safe read
//   - Quit + Service.Stop teardown
//
// Run (with drivers installed, e.g. via vendor/init.go):
//
//	go test -mod=mod -count=1 -timeout=5m -run TestSmokeChrome
func TestSmokeChrome(t *testing.T) {
	if *useDocker {
		t.Skip("Skipping smoke tests under --docker (use host ChromeDriver path)")
	}

	driverPath := *chromeDriverPath
	if driverPath == "" {
		driverPath = findBestPath("vendor/chromedriver*" /*binary=*/, true)
	}
	if driverPath == "" {
		if p, err := exec.LookPath("chromedriver"); err == nil {
			driverPath = p
		}
	}
	if driverPath == "" {
		t.Skip("Skipping smoke: chromedriver not found (set -chrome_driver_path or install vendor/chromedriver)")
	}
	if _, err := os.Stat(driverPath); err != nil {
		t.Skipf("Skipping smoke: chromedriver not found at %q", driverPath)
	}

	chromePath := *chromeBinary
	if _, err := os.Stat(chromePath); err != nil {
		if p, err := exec.LookPath(chromePath); err == nil {
			chromePath = p
		} else if p, err := exec.LookPath("google-chrome"); err == nil {
			chromePath = p
		} else if p, err := exec.LookPath("chromium"); err == nil {
			chromePath = p
		} else if p, err := exec.LookPath("chromium-browser"); err == nil {
			chromePath = p
		} else {
			t.Skipf("Skipping smoke: Chrome binary not found (set -chrome_binary, tried %q)", *chromeBinary)
		}
	}

	port, err := pickUnusedPort()
	if err != nil {
		t.Fatalf("pickUnusedPort: %v", err)
	}

	var opts []selenium.ServiceOption
	if testing.Verbose() {
		selenium.SetDebug(true)
		opts = append(opts, selenium.Output(os.Stderr))
	}
	if *startFrameBuffer {
		opts = append(opts, selenium.StartFrameBuffer())
	}

	svc, err := selenium.NewChromeDriverService(driverPath, port, opts...)
	if err != nil {
		t.Fatalf("NewChromeDriverService(%q, %d): %v", driverPath, port, err)
	}
	defer func() {
		if err := svc.Stop(); err != nil {
			t.Errorf("Service.Stop: %v", err)
		}
	}()

	// Local fixture page (no external network).
	const page = `<!DOCTYPE html>
<html><head><title>smoke</title></head>
<body>
  <h1 class="headline">Hello Smoke</h1>
  <input id="q" name="q" type="text" value="" />
  <button id="go" class="btn primary" type="button">Go</button>
</body></html>`
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(page))
	}))
	defer hs.Close()

	caps := selenium.Capabilities{"browserName": "chrome"}
	chrCaps := chrome.Capabilities{
		Path: chromePath,
		Args: []string{"--no-sandbox", "--disable-dev-shm-usage"},
		W3C:  true,
	}
	if *headless {
		chrCaps.Args = append(chrCaps.Args, "--headless=new")
	}
	caps.AddChrome(chrCaps)

	// ChromeDriver root URL — not /wd/hub (ChromeDriver 115+).
	addr := fmt.Sprintf("http://127.0.0.1:%d", port)
	wd, err := selenium.NewRemote(caps, addr)
	if err != nil {
		t.Fatalf("NewRemote: %v", err)
	}
	defer func() {
		if err := wd.Quit(); err != nil {
			t.Errorf("Quit: %v", err)
		}
	}()

	if err := wd.Get(hs.URL); err != nil {
		t.Fatalf("Get(%s): %v", hs.URL, err)
	}

	// CurrentURL (nil-safe path).
	cur, err := wd.CurrentURL()
	if err != nil {
		t.Fatalf("CurrentURL: %v", err)
	}
	if !strings.HasPrefix(cur, hs.URL) {
		t.Fatalf("CurrentURL = %q, want prefix %q", cur, hs.URL)
	}

	// ByID → W3C CSS mapping.
	input, err := wd.FindElement(selenium.ByID, "q")
	if err != nil {
		t.Fatalf("FindElement(ByID, q): %v", err)
	}

	// ByClassName → W3C [class~=] mapping.
	if _, err := wd.FindElement(selenium.ByClassName, "headline"); err != nil {
		t.Fatalf("FindElement(ByClassName, headline): %v", err)
	}

	// SendKeys (W3C text + value list). Focus first so headless Chrome accepts keys.
	const typed = "smoke-ok"
	if err := input.Click(); err != nil {
		t.Fatalf("Click input: %v", err)
	}
	if err := input.Clear(); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	if err := input.SendKeys(typed); err != nil {
		t.Fatalf("SendKeys: %v", err)
	}
	// DOM property "value" is the reliable read-back (see testGetProperty).
	got, err := input.GetProperty("value")
	if err != nil || got != typed {
		time.Sleep(300 * time.Millisecond)
		got, err = input.GetProperty("value")
	}
	if err != nil {
		t.Fatalf("GetProperty(value): %v", err)
	}
	if got != typed {
		// Last resort: attribute (some builds only expose one of the two).
		if attr, aerr := input.GetAttribute("value"); aerr == nil && attr == typed {
			got = attr
		} else {
			t.Fatalf("input value = %q, want %q", got, typed)
		}
	}

	// Cookie add without expiry/path zero-values breaking the remote end.
	if err := wd.AddCookie(&selenium.Cookie{Name: "smoke", Value: "1"}); err != nil {
		t.Fatalf("AddCookie: %v", err)
	}
	cookies, err := wd.GetCookies()
	if err != nil {
		t.Fatalf("GetCookies: %v", err)
	}
	found := false
	for _, c := range cookies {
		if c.Name == "smoke" && c.Value == "1" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("cookie smoke=1 not found in %#v", cookies)
	}

	title, err := wd.Title()
	if err != nil {
		t.Fatalf("Title: %v", err)
	}
	if title != "smoke" {
		t.Fatalf("Title = %q, want smoke", title)
	}

	// CDP via ChromeDriver (goog/cdp/execute).
	ver, err := wd.ExecuteCDPCommand("Browser.getVersion", nil)
	if err != nil {
		t.Fatalf("ExecuteCDPCommand(Browser.getVersion): %v", err)
	}
	vm, ok := ver.(map[string]interface{})
	if !ok || vm["product"] == nil {
		t.Fatalf("Browser.getVersion unexpected result: %#v", ver)
	}
	product, _ := vm["product"].(string)
	if product == "" {
		t.Fatalf("Browser.getVersion empty product: %#v", ver)
	}
	t.Logf("CDP Browser.getVersion product=%s", product)
}
