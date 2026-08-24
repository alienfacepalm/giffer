package ui_test

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/AlienFacepalm/giffer/internal/ui"
	"github.com/chromedp/chromedp"
)

func TestE2EResetRestoresDefaults(t *testing.T) {
	browser := findBrowser()
	if browser == "" {
		t.Skip("no Chrome/Edge found for e2e reset test")
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

	ctx, cancel := chromedp.NewContext(allocCtx)
	t.Cleanup(cancel)
	ctx, cancel = context.WithTimeout(ctx, 45*time.Second)
	t.Cleanup(cancel)

	var after string
	err := chromedp.Run(ctx,
		chromedp.Navigate(ts.URL+"/"),
		chromedp.WaitReady("#reset-btn"),
		chromedp.Evaluate(`(() => {
			document.getElementById("delay-ms").value = "250";
			document.getElementById("max-width").value = "640";
			document.getElementById("loop").value = "3";
			document.getElementById("drop-label").textContent = "photos.zip";
			document.getElementById("status").hidden = false;
			document.getElementById("status").textContent = "Ready - out.gif";
			document.getElementById("result").classList.remove("is-empty");
			document.getElementById("reset-btn").click();
			return JSON.stringify({
				delay: document.getElementById("delay-ms").value,
				maxWidth: document.getElementById("max-width").value,
				loop: document.getElementById("loop").value,
				label: document.getElementById("drop-label").textContent,
				statusHidden: document.getElementById("status").hidden,
				empty: document.getElementById("result").classList.contains("is-empty"),
				formResetType: typeof document.getElementById("form").reset,
			});
		})()`, &after),
	)
	if err != nil {
		t.Fatalf("chromedp: %v", err)
	}
	if after != `{"delay":"100","maxWidth":"0","loop":"0","label":"Photo archive","statusHidden":true,"empty":true,"formResetType":"function"}` {
		t.Fatalf("reset did not restore baseline: %s", after)
	}
}
