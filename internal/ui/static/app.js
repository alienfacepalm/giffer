(() => {
  const form = document.getElementById("form");
  const fileInput = document.getElementById("file");
  const drop = document.getElementById("drop");
  const dropLabel = document.getElementById("drop-label");
  const go = document.getElementById("go");
  const resetBtn = document.getElementById("reset");
  const status = document.getElementById("status");
  const progress = document.getElementById("progress");
  const progressBar = document.getElementById("progress-bar");
  const progressLabel = document.getElementById("progress-label");
  const progressPct = document.getElementById("progress-pct");
  const result = document.getElementById("result");
  const preview = document.getElementById("preview");
  const download = document.getElementById("download");
  const forge = document.getElementById("forge");
  const forgeCanvas = document.getElementById("forge-canvas");
  const forgeCaption = document.getElementById("forge-caption");

  /** @type {AbortController | null} */
  let convertAbort = null;

  const stageLabel = {
    reading: "Reading frames",
    encoding: "Encoding frames",
    writing: "Writing GIF",
  };

  function startForge() {
    if (!forge || !window.GifferForge) return;
    result.classList.add("is-forging");
    forge.hidden = false;
    forge.setAttribute("aria-hidden", "false");
    window.GifferForge.start(forge, {
      canvas: forgeCanvas,
      captionEl: forgeCaption,
      progress: 0,
    });
  }

  function updateForge(pct, label) {
    if (window.GifferForge) window.GifferForge.setProgress(pct, label);
  }

  function stopForge() {
    if (window.GifferForge) window.GifferForge.stop();
    if (forge) {
      forge.hidden = true;
      forge.setAttribute("aria-hidden", "true");
    }
    result.classList.remove("is-forging");
  }

  /** Relative luminance (0–1) for sRGB channels 0–255. */
  function relativeLuminance(r, g, b) {
    const lin = [r, g, b].map((c) => {
      const s = c / 255;
      return s <= 0.03928 ? s / 12.92 : Math.pow((s + 0.055) / 1.055, 2.4);
    });
    return 0.2126 * lin[0] + 0.7152 * lin[1] + 0.0722 * lin[2];
  }

  function parseCSSColor(input) {
    if (!input) return null;
    const str = String(input).trim();
    const hex = /^#([0-9a-f]{3}|[0-9a-f]{6})$/i.exec(str);
    if (hex) {
      let h = hex[1];
      if (h.length === 3) h = h.split("").map((c) => c + c).join("");
      return {
        r: parseInt(h.slice(0, 2), 16),
        g: parseInt(h.slice(2, 4), 16),
        b: parseInt(h.slice(4, 6), 16),
      };
    }
    const rgb = /^rgba?\(\s*([\d.]+)\s*,\s*([\d.]+)\s*,\s*([\d.]+)/i.exec(str);
    if (rgb) {
      return { r: +rgb[1], g: +rgb[2], b: +rgb[3] };
    }
    return null;
  }

  function contrastRatio(lumA, lumB) {
    const light = Math.max(lumA, lumB);
    const dark = Math.min(lumA, lumB);
    return (light + 0.05) / (dark + 0.05);
  }

  /** Pick ink that stays readable on a background of the given luminance. */
  function readableInk(bgLum) {
    return bgLum < 0.45 ? "#f5f7fa" : "#12151a";
  }

  /** Choose light|dark contrast scheme from a background color or luminance. */
  function contrastSchemeForBackground(bg) {
    let lum;
    if (typeof bg === "number") {
      lum = bg;
    } else {
      const parsed = typeof bg === "string" ? parseCSSColor(bg) : bg;
      if (!parsed) return "light";
      lum = relativeLuminance(parsed.r, parsed.g, parsed.b);
    }
    return lum < 0.45 ? "dark" : "light";
  }

  function preferredContrast() {
    return window.matchMedia("(prefers-color-scheme: dark)").matches
      ? "dark"
      : "light";
  }

  function applyContrast(scheme) {
    const next = scheme === "dark" ? "dark" : "light";
    document.documentElement.setAttribute("data-contrast", next);
    return next;
  }

  /**
   * Keep brand/body text readable: prefer OS scheme, then correct if the
   * resolved body background would make the current ink fail WCAG AA (~4.5).
   */
  function ensureReadableContrast() {
    let scheme = preferredContrast();
    applyContrast(scheme);

    const styles = getComputedStyle(document.body);
    const bg = parseCSSColor(styles.backgroundColor);
    const ink = parseCSSColor(styles.color);
    if (!bg || !ink) return scheme;

    const bgLum = relativeLuminance(bg.r, bg.g, bg.b);
    const inkLum = relativeLuminance(ink.r, ink.g, ink.b);
    if (contrastRatio(bgLum, inkLum) < 4.5) {
      scheme = contrastSchemeForBackground(bgLum);
      applyContrast(scheme);
    }
    return scheme;
  }

  // Expose for e2e / debugging.
  window.__gifferContrast = {
    relativeLuminance,
    parseCSSColor,
    contrastRatio,
    readableInk,
    contrastSchemeForBackground,
    preferredContrast,
    applyContrast,
    ensureReadableContrast,
  };

  ensureReadableContrast();
  if (window.matchMedia) {
    const mq = window.matchMedia("(prefers-color-scheme: dark)");
    const onScheme = () => ensureReadableContrast();
    if (mq.addEventListener) mq.addEventListener("change", onScheme);
    else if (mq.addListener) mq.addListener(onScheme);
  }


  const archiveSuffixes = [
    ".zip",
    ".tar",
    ".tar.gz",
    ".tgz",
    ".tar.bz2",
    ".tbz2",
    ".tbz",
    ".tar.xz",
    ".txz",
    ".7z",
  ];

  function isPhotoArchiveName(name) {
    const lower = String(name || "").toLowerCase();
    return archiveSuffixes.some((suf) => lower.endsWith(suf));
  }

  function validateArchiveFile(file) {
    if (!file) {
      return "Choose a photo archive first (a zip/tar/7z of JPEG, PNG, WebP, or GIF images).";
    }
    if (!isPhotoArchiveName(file.name)) {
      return (
        '"' +
        file.name +
        '" is not a photo archive. Zip your photos (JPEG, PNG, WebP, or GIF) into a .zip and upload that - not audio, video, or loose files.'
      );
    }
    // Client cannot peek inside the zip; server rejects non-photo contents with a fix-it message.
    return "";
  }

  function friendlyError(err) {
    const raw = String((err && err.message) || err || "");
    const msg = raw.trim();
    if (!msg || /failed to fetch|networkerror|load failed|network request failed/i.test(msg)) {
      return (
        "Could not reach the converter (connection dropped). Keep giffer ui running, then upload a zip of photos only " +
        "(JPEG, PNG, WebP, or GIF) - not MP3s or other files - and try again."
      );
    }
    if (/unexpected end of json|json.parse|is not valid json/i.test(msg)) {
      return (
        "The converter returned an incomplete response. Retry with a smaller zip that contains only photos " +
        "(JPEG, PNG, WebP, or GIF)."
      );
    }
    return msg;
  }

  function asNumber(value, fallback) {
    const n = Number(value);
    return Number.isFinite(n) ? n : fallback;
  }

  /** Never put undefined/null/NaN into visible UI copy. */
  function safeLabel(parts) {
    return parts
      .map((p) => {
        if (p == null) return "";
        if (typeof p === "number" && !Number.isFinite(p)) return "";
        const s = String(p);
        if (s === "undefined" || s === "null" || s === "NaN") return "";
        return s;
      })
      .filter(Boolean)
      .join("");
  }

  function setStatus(text, kind) {
    const msg = text == null ? "" : String(text);
    const clean =
      msg === "undefined" || msg === "null" || msg === "NaN" ? "" : msg;
    status.hidden = !clean;
    status.textContent = clean;
    status.classList.toggle("is-error", kind === "error");
    status.classList.toggle("is-ok", kind === "ok");
  }

  function setProgress(pct, label) {
    const n = Math.max(0, Math.min(100, asNumber(pct, 0) | 0));
    progress.hidden = false;
    progressBar.value = n;
    progressPct.textContent = safeLabel([n, "%"]);
    if (label) progressLabel.textContent = safeLabel([label]);
  }

  function hideProgress() {
    progress.hidden = true;
    progressBar.value = 0;
    progressPct.textContent = "0%";
    progressLabel.textContent = "Converting…";
  }

  function setFileName(name) {
    dropLabel.textContent = name || "Drop a photo archive";
  }

  function clearPreview() {
    stopForge();
    result.classList.add("is-empty");
    preview.hidden = true;
    download.hidden = true;
    preview.removeAttribute("src");
    download.removeAttribute("href");
  }

  /** Restore file, fields, progress, status, and preview to first-load defaults. */
  function resetToBaseline() {
    if (convertAbort) {
      convertAbort.abort();
      convertAbort = null;
    }
    form.reset();
    fileInput.value = "";
    drop.classList.remove("is-drag");
    setFileName("");
    setStatus("");
    hideProgress();
    clearPreview();
    go.disabled = false;
    if (resetBtn) resetBtn.disabled = false;
  }

  function showPreview(url, filename) {
    stopForge();
    preview.src = url;
    preview.hidden = false;
    download.href = url;
    download.download = filename || "out.gif";
    download.hidden = false;
    result.classList.remove("is-empty");
  }

  ["dragenter", "dragover"].forEach((evt) => {
    drop.addEventListener(evt, (e) => {
      e.preventDefault();
      drop.classList.add("is-drag");
    });
  });
  ["dragleave", "drop"].forEach((evt) => {
    drop.addEventListener(evt, (e) => {
      e.preventDefault();
      drop.classList.remove("is-drag");
    });
  });
  drop.addEventListener("drop", (e) => {
    const files = e.dataTransfer?.files;
    if (!files?.length) return;
    fileInput.files = files;
    setFileName(files[0].name);
    const bad = validateArchiveFile(files[0]);
    if (bad) setStatus(bad, "error");
    else setStatus("");
  });
  fileInput.addEventListener("change", () => {
    const file = fileInput.files?.[0];
    setFileName(file?.name || "");
    const bad = validateArchiveFile(file);
    if (bad) setStatus(bad, "error");
    else setStatus("");
  });

  async function readConvertStream(res) {
    const ct = res.headers.get("content-type") || "";
    if (!ct.includes("ndjson") || !res.body) {
      const data = await res.json();
      if (!res.ok || data.error) {
        throw new Error(data.error || "Conversion failed.");
      }
      return data;
    }

    const reader = res.body.getReader();
    const decoder = new TextDecoder();
    let buf = "";
    let final = null;

    while (true) {
      const { value, done } = await reader.read();
      if (done) break;
      buf += decoder.decode(value, { stream: true });
      const lines = buf.split("\n");
      buf = lines.pop() || "";
      for (const line of lines) {
        const trimmed = line.trim();
        if (!trimmed) continue;
        let ev;
        try {
          ev = JSON.parse(trimmed);
        } catch {
          throw new Error("Conversion failed: incomplete response from server.");
        }
        if (ev.type === "progress") {
          const stage = stageLabel[ev.stage] || "Converting";
          const done = asNumber(ev.done, 0);
          const total = asNumber(ev.total, 0);
          const pct = asNumber(ev.percent, 0);
          const detail =
            total > 0
              ? safeLabel([stage, " (", done, "/", total, ")"])
              : stage;
          setProgress(pct, detail);
          updateForge(pct, detail);
        } else if (ev.type === "error") {
          throw new Error(ev.error || "Conversion failed.");
        } else if (ev.type === "done") {
          setProgress(100, "Done");
          updateForge(100, "Sealing the GIF…");
          final = ev;
        }
      }
    }

    if (buf.trim()) {
      const ev = JSON.parse(buf.trim());
      if (ev.type === "error") throw new Error(ev.error || "Conversion failed.");
      if (ev.type === "done") final = ev;
    }

    if (!final || !final.ok) {
      throw new Error("Conversion failed.");
    }
    return final;
  }

  if (resetBtn) {
    resetBtn.addEventListener("click", () => {
      resetToBaseline();
    });
  }

  form.addEventListener("submit", async (e) => {
    e.preventDefault();
    clearPreview();
    setStatus("");
    hideProgress();

    const picked = fileInput.files?.[0];
    const bad = validateArchiveFile(picked);
    if (bad) {
      setStatus(bad, "error");
      return;
    }

    const body = new FormData();
    body.set("file", picked);
    body.set("delay-ms", document.getElementById("delay-ms").value);
    body.set("max-width", document.getElementById("max-width").value);
    body.set("loop", document.getElementById("loop").value);

    if (convertAbort) convertAbort.abort();
    const ac = new AbortController();
    convertAbort = ac;
    const { signal } = ac;

    go.disabled = true;
    setProgress(0, "Uploading…");
    startForge();
    updateForge(0, "Opening the archive…");

    try {
      const res = await fetch("/api/convert", { method: "POST", body, signal });
      if (!res.ok && !(res.headers.get("content-type") || "").includes("ndjson")) {
        const data = await res.json().catch(() => ({}));
        throw new Error(data.error || "Conversion failed.");
      }
      const data = await readConvertStream(res);
      const url = data.url + "?t=" + Date.now();
      const filename = (data.output || "out.gif").split(/[/\\]/).pop();
      showPreview(url, filename);
      setStatus("Ready - " + data.output, "ok");
    } catch (err) {
      if (err && err.name === "AbortError") {
        return;
      }
      hideProgress();
      clearPreview();
      setStatus(friendlyError(err), "error");
    } finally {
      if (convertAbort === ac) convertAbort = null;
      go.disabled = false;
    }
  });
})();
