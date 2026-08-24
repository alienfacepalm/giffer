package ui_test

import (
	"context"
	"fmt"
	"net/http/httptest"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/AlienFacepalm/giffer/internal/ui"
	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/chromedp"
)

type contrastProbe struct {
	Scheme  string  `json:"scheme"`
	Ratio   float64 `json:"ratio"`
	DropGap float64 `json:"dropGap"`
	OK      bool    `json:"ok"`
}

func TestE2EReadableContrast(t *testing.T) {
	browser := findBrowser()
	if browser == "" {
		t.Skip("no Chrome/Edge found for e2e contrast tests")
	}

	srv := ui.New(ui.Options{UploadDir: t.TempDir()})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(),
		append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.ExecPath(browser),
			chromedp.Flag("headless", true),
			chromedp.Flag("disable-gpu", true),
			chromedp.Flag("no-sandbox", true),
			chromedp.Flag("disable-dev-shm-usage", true),
		)...,
	)
	t.Cleanup(allocCancel)

	viewports := []struct {
		name   string
		width  int64
		height int64
		mobile bool
	}{
		{name: "desktop", width: 1200, height: 800, mobile: false},
		{name: "tablet", width: 760, height: 900, mobile: false},
		{name: "mobile", width: 390, height: 844, mobile: true},
	}
	schemes := []string{"light", "dark"}

	for _, vp := range viewports {
		for _, scheme := range schemes {
			t.Run(fmt.Sprintf("%s/%s", vp.name, scheme), func(t *testing.T) {
				ctx, cancel := chromedp.NewContext(allocCtx)
				t.Cleanup(cancel)
				ctx, cancel = context.WithTimeout(ctx, 45*time.Second)
				t.Cleanup(cancel)

				var probe contrastProbe
				err := chromedp.Run(ctx,
					emulation.SetDeviceMetricsOverride(vp.width, vp.height, 1, vp.mobile),
					emulation.SetEmulatedMedia().WithFeatures([]*emulation.MediaFeature{
						{Name: "prefers-color-scheme", Value: scheme},
					}),
					chromedp.Navigate(ts.URL+"/"),
					chromedp.WaitReady("body"),
					chromedp.Evaluate(`window.__gifferContrast.ensureReadableContrast()`, nil),
					chromedp.Evaluate(`(() => {
						const api = window.__gifferContrast;
						const mark = document.querySelector(".mark");
						const drop = document.getElementById("drop");
						const fields = document.querySelector(".fields");
						const bg = api.parseCSSColor(getComputedStyle(document.body).backgroundColor);
						const ink = api.parseCSSColor(getComputedStyle(mark).color);
						const bgLum = api.relativeLuminance(bg.r, bg.g, bg.b);
						const inkLum = api.relativeLuminance(ink.r, ink.g, ink.b);
						const ratio = api.contrastRatio(bgLum, inkLum);
						const want = api.parseCSSColor(api.readableInk(bgLum));
						const close = Math.abs(ink.r - want.r) < 40
							&& Math.abs(ink.g - want.g) < 40
							&& Math.abs(ink.b - want.b) < 40;
						const dropGap = fields.getBoundingClientRect().top - drop.getBoundingClientRect().bottom;
						return {
							scheme: document.documentElement.getAttribute("data-contrast"),
							ratio,
							dropGap,
							ok: ratio >= 4.5 && close,
						};
					})()`, &probe),
				)
				if err != nil {
					t.Fatalf("chromedp: %v", err)
				}
				if probe.Scheme != scheme {
					t.Fatalf("data-contrast=%q, want %q", probe.Scheme, scheme)
				}
				if !probe.OK {
					t.Fatalf("brand text not readable (ratio=%.2f scheme=%s)", probe.Ratio, probe.Scheme)
				}
				if probe.DropGap < 12 {
					t.Fatalf("drop→fields gap too small: %.1fpx (want ≥12)", probe.DropGap)
				}
			})
		}
	}
}

func TestE2ETextFlipsWhenBackgroundGoesDark(t *testing.T) {
	browser := findBrowser()
	if browser == "" {
		t.Skip("no Chrome/Edge found for e2e contrast tests")
	}

	srv := ui.New(ui.Options{UploadDir: t.TempDir()})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(),
		append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.ExecPath(browser),
			chromedp.Flag("headless", true),
			chromedp.Flag("disable-gpu", true),
			chromedp.Flag("no-sandbox", true),
		)...,
	)
	t.Cleanup(allocCancel)

	ctx, cancel := chromedp.NewContext(allocCtx)
	t.Cleanup(cancel)
	ctx, cancel = context.WithTimeout(ctx, 45*time.Second)
	t.Cleanup(cancel)

	var before string
	var after struct {
		Scheme string `json:"scheme"`
		OK     bool   `json:"ok"`
	}

	err := chromedp.Run(ctx,
		emulation.SetEmulatedMedia().WithFeatures([]*emulation.MediaFeature{
			{Name: "prefers-color-scheme", Value: "light"},
		}),
		chromedp.Navigate(ts.URL+"/"),
		chromedp.WaitReady("body"),
		chromedp.Evaluate(`window.__gifferContrast.ensureReadableContrast()`, nil),
		chromedp.Evaluate(`document.documentElement.getAttribute("data-contrast")`, &before),
		chromedp.Evaluate(`(() => {
			const api = window.__gifferContrast;
			const scheme = api.contrastSchemeForBackground("#12151a");
			api.applyContrast(scheme);
			const ink = api.parseCSSColor(getComputedStyle(document.querySelector(".mark")).color);
			const want = api.parseCSSColor(api.readableInk(api.relativeLuminance(0x12, 0x15, 0x1a)));
			const close = Math.abs(ink.r - want.r) < 40
				&& Math.abs(ink.g - want.g) < 40
				&& Math.abs(ink.b - want.b) < 40;
			return {
				scheme: document.documentElement.getAttribute("data-contrast"),
				ok: scheme === "dark" && close,
			};
		})()`, &after),
	)
	if err != nil {
		t.Fatalf("chromedp: %v", err)
	}
	if before != "light" {
		t.Fatalf("expected starting light contrast, got %q", before)
	}
	if !after.OK || after.Scheme != "dark" {
		t.Fatalf("expected readable light ink after dark bg flip; got scheme=%q ok=%v", after.Scheme, after.OK)
	}
}

func TestE2EContrastHelpers(t *testing.T) {
	browser := findBrowser()
	if browser == "" {
		t.Skip("no Chrome/Edge found for e2e contrast tests")
	}

	srv := ui.New(ui.Options{UploadDir: t.TempDir()})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(),
		append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.ExecPath(browser),
			chromedp.Flag("headless", true),
			chromedp.Flag("disable-gpu", true),
			chromedp.Flag("no-sandbox", true),
		)...,
	)
	t.Cleanup(allocCancel)

	ctx, cancel := chromedp.NewContext(allocCtx)
	t.Cleanup(cancel)
	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	t.Cleanup(cancel)

	var checks map[string]bool
	err := chromedp.Run(ctx,
		chromedp.Navigate(ts.URL+"/"),
		chromedp.WaitReady("body"),
		chromedp.Evaluate(`(() => {
			const api = window.__gifferContrast;
			return {
				darkBgWantsLightInk: api.readableInk(0.05) === "#f5f7fa",
				lightBgWantsDarkInk: api.readableInk(0.9) === "#12151a",
				darkScheme: api.contrastSchemeForBackground("#0f1216") === "dark",
				lightScheme: api.contrastSchemeForBackground("#e8edf2") === "light",
				ratioPass: api.contrastRatio(0.05, 0.95) >= 4.5,
				ratioFail: api.contrastRatio(0.5, 0.55) < 4.5,
			};
		})()`, &checks),
	)
	if err != nil {
		t.Fatalf("chromedp: %v", err)
	}
	for name, pass := range checks {
		if !pass {
			t.Fatalf("helper check failed: %s (all=%v)", name, checks)
		}
	}
}

func findBrowser() string {
	candidates := []string{
		os.Getenv("GIFFER_E2E_BROWSER"),
		os.Getenv("CHROME_PATH"),
	}
	if runtime.GOOS == "windows" {
		candidates = append(candidates,
			`C:\Program Files\Google\Chrome\Application\chrome.exe`,
			`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
			`C:\Program Files\Microsoft\Edge\Application\msedge.exe`,
			`C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`,
		)
	} else {
		candidates = append(candidates,
			"/usr/bin/google-chrome",
			"/usr/bin/chromium",
			"/usr/bin/chromium-browser",
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		)
	}
	for _, p := range candidates {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	return ""
}
