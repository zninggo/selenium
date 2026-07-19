# Go 版 Selenium / WebDriver 客户端（维护中的 fork）

**语言：** [English](README.md) | **简体中文**

| | |
|---|---|
| **模块路径** | `github.com/zninggo/selenium` |
| **上游仓库** | [tebeka/selenium](https://github.com/tebeka/selenium)（约 2021 年后基本停更） |
| **状态** | 先自用，公开 best-effort |
| **当前版本** | `v0.11.1` |
| **相对上游的主要差异** | ChromeDriver 115+ / W3C 定位与 SendKeys / Selenium 4 服务 / CDP / HTTP 可靠性 / 多平台驱动下载 / smoke 与可选浏览器 CI |

[![Go Reference](https://pkg.go.dev/badge/github.com/zninggo/selenium.svg)](https://pkg.go.dev/github.com/zninggo/selenium)
[![CI](https://github.com/zninggo/selenium/actions/workflows/ci.yml/badge.svg)](https://github.com/zninggo/selenium/actions/workflows/ci.yml)

这是 Go 语言的 [WebDriver][webdriver] 客户端，兼容 [ChromeDriver][chromedriver]、[Geckodriver][geckodriver] 以及 [Selenium][selenium] 3/4 服务端。

本仓库是 [tebeka/selenium](https://github.com/tebeka/selenium) 的维护性 fork。可按需 cherry-pick 上游提交，**不会**自动整并 `upstream/master`。

[selenium]: https://www.selenium.dev/
[webdriver]: https://www.w3.org/TR/webdriver/
[go]: https://go.dev/
[geckodriver]: https://github.com/mozilla/geckodriver
[chromedriver]: https://googlechromelabs.github.io/chrome-for-testing/

## 安装

```text
go get github.com/zninggo/selenium@latest
```

需要 **Go 1.22+**，以及可用的 WebDriver 栈（浏览器 + driver，或 Selenium Server）。

### 从 tebeka/selenium 迁移

1. 修改 import：`github.com/tebeka/selenium` → `github.com/zninggo/selenium`
2. `go get github.com/zninggo/selenium@v0.11.1`
3. `go mod tidy`

## 快速开始（推荐）

### ChromeDriver 直连（ChromeDriver 115+）

ChromeDriver 在服务端 **根路径** 提供 WebDriver 接口——**不要**再加 `/wd/hub`。

```go
package main

import (
	"fmt"

	"github.com/zninggo/selenium"
	"github.com/zninggo/selenium/chrome"
)

func main() {
	const (
		chromeDriverPath = "chromedriver" // 或 vendor/init.go 下载后的路径
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

	// 根路径 —— 不是 http://127.0.0.1:9515/wd/hub
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

完整示例见 [example_test.go](example_test.go) 中的 `ExampleChromeDriver`。

### Selenium 4 standalone

```go
svc, err := selenium.NewSeleniumServiceV4("selenium-server-4.x.jar", 4444)
// ...
wd, err := selenium.NewRemote(caps, "http://127.0.0.1:4444") // 根路径，无 /wd/hub
```

见 `Example_selenium4`（[example_test.go](example_test.go)）。

### Selenium 3（遗留）

`NewSeleniumService` 仍面向 Selenium 3（`GridLauncherV3`），监听在 **`/wd/hub`** 下：

```go
svc, err := selenium.NewSeleniumService("selenium-server.jar", 4444)
wd, err := selenium.NewRemote(caps, "http://127.0.0.1:4444/wd/hub")
```

见原有 `Example`（[example_test.go](example_test.go)）。

### URL 前缀速查

| 后端 | 启动函数 | `NewRemote` 的 urlPrefix |
|------|----------|---------------------------|
| ChromeDriver 115+ | `NewChromeDriverService` | `http://127.0.0.1:<port>` |
| Geckodriver | `NewGeckoDriverService` | `http://127.0.0.1:<port>` |
| Selenium 4 | `NewSeleniumServiceV4` | `http://127.0.0.1:<port>` |
| Selenium 3 | `NewSeleniumService` | `http://127.0.0.1:<port>/wd/hub` |

## 本 fork 行为说明

| 主题 | 行为 |
|------|------|
| HTTP 客户端 | 默认 `HTTPClient` 超时 **120s**（可整体替换 `selenium.HTTPClient`） |
| 请求头 | 带 JSON body 时设置 `Content-Type: application/json` |
| Cookie | 可选字段带 `omitempty`（零值的 `Expiry`/`Path`/… 不会序列化） |
| W3C 定位 | W3C 模式下 `ByID` / `ByName` / `ByClassName` 映射为 CSS |
| W3C SendKeys | 同时发送 `text` 与 `value` 字符串列表 |
| 服务调试 | 仅当 `SetDebug(true)` 时，Selenium 3 才加 `-debug` |
| Linux 清理 | 进程组 + `Pdeathsig`，减少孤儿 driver 进程 |
| CDP | `ExecuteCDPCommand(cmd, params)`，经 ChromeDriver `goog/cdp/execute` |

## 下载依赖（测试用）

主要用于本地测试 / smoke：

```text
$ cd vendor
$ go run init.go --alsologtostderr --download_browsers --download_latest
$ cd ..
```

按 `GOOS`/`GOARCH` 选择资产（linux64、mac-x64、mac-arm64、win32、win64）。
Chrome 来自 [Chrome for Testing](https://googlechromelabs.github.io/chrome-for-testing/) Stable。
macOS/Windows 上的 Firefox 可能是安装包/DMG，需手动安装。

## 文档

- API：https://pkg.go.dev/github.com/zninggo/selenium
- 示例：[example_test.go](example_test.go)
- 主路径 smoke：`TestSmokeChrome`（[smoke_test.go](smoke_test.go)）
- 英文 README：[README.md](README.md)

## Smoke 测试（可选）

需要本机 Chrome + ChromeDriver。缺失时会 **skip**（默认 CI 仍可绿；若 runner 已有 Chrome 则可能真正执行）。

```text
cd vendor && go run init.go --alsologtostderr --download_browsers && cd ..
go test -mod=mod -count=1 -timeout=5m -run TestSmokeChrome -v
```

## 已知问题

很多失败来自浏览器/driver，而非本客户端。若客户端行为异常，请[提 issue](https://github.com/zninggo/selenium/issues/new)。

### Selenium 2

不再支持。

### Selenium 3

1. [Selenium 3 NewSession 未完整实现 W3C 参数](https://github.com/SeleniumHQ/selenium/issues/2827)。

### Geckodriver（独立）

1. [不支持 Log API](https://github.com/mozilla/geckodriver/issues/284)。
2. [点击相关问题](https://github.com/mozilla/geckodriver/issues/1007)。
3. [控制类按键可能需要结尾 null key](https://github.com/mozilla/geckodriver/issues/665)。

### Chromedriver

1. [无头 Chrome 不支持扩展](https://crbug.com/706008)。
2. ChromeDriver 115+ 请使用 **根路径** 服务 URL（不要 `/wd/hub`）。

## 更新摘要

| 版本 | 要点 |
|------|------|
| v0.10.0 | fork 迁址、Go 1.22、GitHub Actions |
| v0.10.1 | HTTP body Close、客户端超时、CurrentURL、cookie expiry |
| v0.10.2 | W3C class/name 定位、`NewSeleniumServiceV4` |
| v0.10.3 | Linux 进程组 / Pdeathsig 孤儿清理 |
| v0.10.4 | 多平台 `vendor/init.go`、Chrome for Testing |
| v0.10.5 | Content-Type、W3C SendKeys value 列表、cookie omitempty、按需 `-debug` |
| v0.10.6 | `TestSmokeChrome`；ChromeDriver 测试去掉 `/wd/hub` |
| v0.10.7 | headless CI 上用 `GetProperty` 读回输入 |
| v0.10.8 | README 现代用法；`ExampleChromeDriver` / `Example_selenium4` |
| v0.11.0 | `ExecuteCDPCommand`；可选 `browser` 工作流 |
| v0.11.1 | 中文 README（`README.zh-CN.md`）与语言切换 |

完整记录见 [ChangeLog](ChangeLog) 与 [Releases](https://github.com/zninggo/selenium/releases)。

### 可选浏览器 CI

手动运行（或每周定时）**browser** 工作流：  
Actions → browser → Run workflow。  
会下载 Chrome for Testing 并运行 `TestSmokeChrome`（含 CDP）。

### 可选后续

1. 功能类 PR（Shadow DOM、Print、Select 封装、远程文件上传）

## 历史破坏性变更

### 2017-08-22

未使用的 `Version` 常量已删除。

### 2017-04-18

`Log` 方法改为接受类型化常量，返回值更符合 Go 习惯。

## 开发

欢迎 PR。请确保：

1. 非琐碎改动有测试，且现有测试通过。
2. 已对改动文件执行 `gofmt`。可选 pre-commit：

```text
ln -s ../../misc/git/pre-commit .git/hooks/pre-commit
```

[Issues](https://github.com/zninggo/selenium/issues)

### 本地测试

```text
sudo apt-get install xvfb openjdk-11-jre   # 如需要
go test -mod=mod ./...
```

缺少二进制时，顶层浏览器相关测试会 skip：

- `TestChrome` / `TestSmokeChrome`
- `TestFirefoxSelenium3` / `TestFirefoxGeckoDriver`
- `TestHTMLUnit`（需要 Java + JAR）

路径可用 `go test` 参数配置（见 `go test -args -help`）。

### Docker 测试

```text
go test --docker
```

或：

```text
docker build -t go-selenium testing/
docker run --volume=$(pwd):/code --workdir=/code -it go-selenium bash
# 容器内：testing/docker-test.sh
```

### Sauce Labs 测试

```text
go test --test.run=TestSauce --test.timeout=20m \
  --experimental_enable_sauce \
  --sauce_user_name=[username] \
  --sauce_access_key=[access key]
```

## 许可证

MIT — 见 [LICENSE](LICENSE)。
