(() => {
  const form = document.getElementById("form");
  const fileInput = document.getElementById("file");
  const drop = document.getElementById("drop");
  const dropLabel = document.getElementById("drop-label");
  const go = document.getElementById("go");
  const status = document.getElementById("status");
  const result = document.getElementById("result");
  const preview = document.getElementById("preview");
  const download = document.getElementById("download");

  function setStatus(text, kind) {
    status.hidden = !text;
    status.textContent = text || "";
    status.classList.toggle("is-error", kind === "error");
    status.classList.toggle("is-ok", kind === "ok");
  }

  function setFileName(name) {
    dropLabel.textContent = name || "Drop a .zip here";
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
  });
  fileInput.addEventListener("change", () => {
    setFileName(fileInput.files?.[0]?.name || "");
  });

  form.addEventListener("submit", async (e) => {
    e.preventDefault();
    result.hidden = true;
    setStatus("");

    if (!fileInput.files?.length) {
      setStatus("Choose a .zip first.", "error");
      return;
    }

    const body = new FormData();
    body.set("file", fileInput.files[0]);
    body.set("delay-ms", document.getElementById("delay-ms").value);
    body.set("max-width", document.getElementById("max-width").value);
    body.set("loop", document.getElementById("loop").value);

    go.disabled = true;
    setStatus("Converting…");

    try {
      const res = await fetch("/api/convert", { method: "POST", body });
      const data = await res.json();
      if (!res.ok || !data.ok) {
        setStatus(data.error || "Conversion failed.", "error");
        return;
      }
      const url = data.url + "?t=" + Date.now();
      preview.src = url;
      download.href = url;
      download.download = (data.output || "out.gif").split(/[/\\]/).pop();
      result.hidden = false;
      setStatus("Ready — " + data.output, "ok");
    } catch (err) {
      setStatus(String(err.message || err), "error");
    } finally {
      go.disabled = false;
    }
  });
})();
