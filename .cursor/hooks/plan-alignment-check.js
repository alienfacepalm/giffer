#!/usr/bin/env node
/**
 * Monitor agent/subagent completions: log changed files and inject a one-shot
 * plan-alignment follow-up when this turn (or subagent) actually edited files.
 *
 * Modes:
 *   --record-edit       afterFileEdit: track paths for this generation
 *   --on-stop           stop: follow-up if tracked edits exist
 *   --on-subagent-stop  subagentStop: follow-up if modified_files exist
 */
const fs = require("node:fs");
const path = require("node:path");

const STATE_DIR = path.join(".cursor", "hooks", "state");
const LOG_PATH = path.join(STATE_DIR, "agent-completions.jsonl");
const PENDING_PATH = path.join(STATE_DIR, "pending-edits.json");

const FOLLOWUP = [
  "Quick plan-alignment check (hook-triggered, one pass only):",
  "1. Diff the changes from this turn against SPEC.md (Phase 1 CLI scope, parameters, non-goals).",
  "2. Check .cursor/rules: Go layout/tests (`internal/convert` vs `internal/cli`), upload/ zip paths, ALL-CAPS .md names.",
  "3. Flag plan deviations, missing SPEC edge-case tests, or premature optimizations outside the Phase 1 surface.",
  "4. Fix only clear violations; if aligned, reply with a short 'aligned' note and stop — do not expand scope.",
].join(" ");

function readStdin() {
  try {
    return fs.readFileSync(0, "utf8");
  } catch {
    return "";
  }
}

function parseInput(raw) {
  try {
    return raw ? JSON.parse(raw) : {};
  } catch (err) {
    console.error("[plan-alignment-check] invalid JSON stdin:", err.message);
    return null;
  }
}

function ensureStateDir() {
  fs.mkdirSync(STATE_DIR, { recursive: true });
}

function readPending() {
  try {
    return JSON.parse(fs.readFileSync(PENDING_PATH, "utf8"));
  } catch {
    return {};
  }
}

function writePending(store) {
  ensureStateDir();
  fs.writeFileSync(PENDING_PATH, JSON.stringify(store, null, 2), "utf8");
}

function pendingKey(input) {
  const conversationId = input.conversation_id || "unknown";
  const generationId = input.generation_id || "unknown";
  return `${conversationId}::${generationId}`;
}

function toRepoRelative(filePath) {
  if (!filePath) return "";
  const abs = path.resolve(filePath);
  const root = process.cwd();
  if (abs.toLowerCase().startsWith(root.toLowerCase() + path.sep)) {
    return path.relative(root, abs).split(path.sep).join("/");
  }
  return filePath.split(path.sep).join("/");
}

function appendLog(entry) {
  try {
    ensureStateDir();
    fs.appendFileSync(LOG_PATH, `${JSON.stringify(entry)}\n`, "utf8");
  } catch (err) {
    console.error("[plan-alignment-check] log failed:", err.message);
  }
}

function emitFollowUp(changedFiles) {
  const fileList = changedFiles.slice(0, 20).join(", ");
  const more =
    changedFiles.length > 20 ? ` (+${changedFiles.length - 20} more)` : "";
  process.stdout.write(
    JSON.stringify({
      followup_message: `${FOLLOWUP} Changed files: ${fileList}${more}.`,
    }) + "\n",
  );
}

function recordEdit(input) {
  const rel = toRepoRelative(input.file_path);
  if (!rel) {
    process.stdout.write("{}\n");
    return;
  }
  const key = pendingKey(input);
  const store = readPending();
  const files = new Set(store[key] || []);
  files.add(rel);
  store[key] = [...files];
  writePending(store);
  process.stdout.write("{}\n");
}

function onStop(input) {
  const status = input.status || "unknown";
  const loopCount = Number(input.loop_count || 0);
  const key = pendingKey(input);
  const store = readPending();
  const changedFiles = store[key] || [];

  appendLog({
    at: new Date().toISOString(),
    event: "stop",
    status,
    loop_count: loopCount,
    conversation_id: input.conversation_id || null,
    generation_id: input.generation_id || null,
    changed_files: changedFiles,
  });

  if (status === "completed" && loopCount === 0 && changedFiles.length > 0) {
    delete store[key];
    writePending(store);
    emitFollowUp(changedFiles);
    return;
  }

  if (changedFiles.length > 0 && (status !== "completed" || loopCount > 0)) {
    delete store[key];
    writePending(store);
  }

  process.stdout.write("{}\n");
}

function onSubagentStop(input) {
  const status = input.status || "unknown";
  const loopCount = Number(input.loop_count || 0);
  const changedFiles = (Array.isArray(input.modified_files)
    ? input.modified_files
    : []
  )
    .map(toRepoRelative)
    .filter(Boolean);

  appendLog({
    at: new Date().toISOString(),
    event: "subagentStop",
    status,
    loop_count: loopCount,
    subagent_type: input.subagent_type || null,
    description: input.description || null,
    task: input.task || null,
    duration_ms: input.duration_ms ?? null,
    conversation_id: input.conversation_id || input.parent_conversation_id || null,
    changed_files: changedFiles,
  });

  if (status === "completed" && loopCount === 0 && changedFiles.length > 0) {
    emitFollowUp(changedFiles);
    return;
  }

  process.stdout.write("{}\n");
}

function main() {
  const mode = process.argv[2] || "--on-stop";
  const raw = readStdin();
  const input = parseInput(raw);
  if (input === null) {
    process.stdout.write("{}\n");
    return;
  }

  if (mode === "--record-edit") {
    recordEdit(input);
    return;
  }
  if (mode === "--on-subagent-stop") {
    onSubagentStop(input);
    return;
  }
  onStop(input);
}

main();
