# MaaS Gateway Dev Console — Agent Guide

This is a standalone local Web console, configurable HTTP/SSE mock server, and
Gateway process supervisor. Read [README.md](README.md) for usage. The matching
human-readable Chinese guide is [AGENTS.zh-CN.md](AGENTS.zh-CN.md).

## 1. Scope and repository boundaries

- This directory is an independent Git repository. Run Git commands here and
  confirm `git rev-parse --show-toplevel` before staging; do not stage sibling projects.
- Do not modify `../maas-gateway-local-mock`; it owns Gateway black-box E2E tests.
- Do not modify, reset, or replace the configured Gateway source checkout.
- Keep normal persistent configuration in `profiles/`; generated session files
  belong in `.runtime/`. `.runtime/`, `.idea/`, logs, and `.DS_Store` are ignored.
- Keep the application dependency-free when the Go standard library and plain
  HTML/CSS/JavaScript suffice. Node.js is optional for frontend tests only.
- Follow the parent workspace's approval rules. Diagnosis and explanation are
  read-only unless the user has authorized the specific implementation.

## 2. Code ownership

| Area | Owner |
| --- | --- |
| CLI, active profile selection, console lifecycle | `cmd/dev-console/main.go` |
| Profile schema, defaults, validation, persistence, source-path expansion | `internal/config/` |
| Mock matching, responses, request recording | `internal/mock/` |
| Generated Gateway YAML | `internal/runtime/gateway_config.go` |
| Gateway/Delve lifecycle, process groups, captured logs | `internal/process/` |
| Local API and embedded frontend assets | `internal/web/` |
| Frontend behavior regression checks | `scripts/ui.test.cjs` |

Prefer the owning module and its immediate callers over repository-wide scans.

## 3. Configuration and startup contract

- Normally start the console from this repository root. The working directory
  determines `.runtime/`; relative profile paths are also resolved from it.
- The default profile is `profiles/default.json`; `-profile <path>` selects another
  file. A missing file is created from defaults. UI saves persist the active profile.
- The default Gateway source expression is `${MAAS_GATEWAY_DIR}`. Resolve it only
  when starting Gateway, using the console process's environment. If that variable
  is unset or empty, use `$HOME/GolandProjects/maas-gateway`.
- Keep variable expressions in saved profiles and forms. Also support explicit
  paths and other environment variables, such as `${HOME}`. An unset variable
  other than `MAAS_GATEWAY_DIR` must produce a visible error, not a guessed path.
- Each Gateway start generates `.runtime/current/gateway-config/config.<ENV>.yml`
  and sets `ENV` and `CONF_PATH` for the child process. Do not read or merge the
  Gateway checkout's original YAML as an implicit configuration source.
- Additional Gateway YAML is appended as text; it is not a deep merge.
  Generated YAML is disposable and will be overwritten on the next start.
- Gateway configuration changes require Apply & Restart. Mock route saves apply
  to subsequent requests immediately. Listener-address changes and changes to
  the console process's environment require a console restart.
- Bind the console, Mock, and Delve to loopback by default. Stop must terminate
  the managed process group, including Go/Delve descendants.

## 4. Mock interfaces and scenarios

- Persist a flat `mock.routes` array. Group routes by Method + Path in the UI only;
  each route remains one independently identified matching/response scenario.
- Match enabled routes by descending Priority; ties preserve profile order.
  Headers, query values, and JSON field conditions combine with AND. Return the
  first match. An empty match is a fallback and should have lower priority.
- Unknown routes must return HTTP 501 and be recorded with `matched=false`.
  Never conceal missing dependencies with a catch-all success response.
- Add scenario and Duplicate create disabled branches. Enable and save them
  after configuring their matching conditions. A new interface's initial route
  is enabled by default, and becomes active only after saving.
- Interface descriptions appear below sidebar entries. The UI propagates a saved
  description to scenarios sharing the same Method + Path.
- Valid editor drafts survive scenario switches in browser memory. Persist only
  through explicit save actions; keep the unsaved indicator and prevent invalid
  JSON or duplicate IDs from silently overwriting another scenario.
- SSE uses the existing `response.sse` array of `{data, delay_ms}` events. Keep
  normal responses, streaming responses, plain-text data, and `[DONE]` compatible.
  `close_after_events=0` sends all events; a positive value simulates early closure.
- The event-card editor supports add/delete/reorder/format and per-event delays.
  Preserve original data until edited; display formatting alone must not rewrite it.

## 5. Frontend and visual contract

- The approved theme is EVA/NERV-inspired: black/charcoal surfaces, orange
  industrial headings, purple scene selection, lime active/Run accents, and red
  destructive/error states. Use clipped corners and restrained hazard stripes.
- Keep fields, code, and logs readable. Preserve bilingual action labels and
  responsive layouts rather than adding decorative elements that obscure controls.
- Implement visual assets with HTML/CSS/SVG. Design images are references, not a
  replacement for working controls. Static assets are embedded into the Go binary;
  restart/rebuild the console and refresh the browser to see frontend changes.
- All selects use the shared custom dark popup. Native select values remain the
  form data source; keep them synchronized with the visible controls.
- Preserve keyboard navigation, Enter selection, Escape/outside-click closure,
  focus indicators, ARIA state, and viewport-aware popup positioning. Highlighting
  an option must scroll the menu only, not the document.
- Color Gateway logs by declared severity: ERROR/FATAL/PANIC red, WARN orange,
  INFO green, DEBUG/TRACE purple. Support JSON and text logs, strip terminal ANSI
  controls, and render user-controlled content with `textContent`, not raw HTML.
- Keep the 5000-entry display limit and automatic follow behavior: reading older
  logs must not force the viewer back to the bottom.

## 6. Validation and local verification

```bash
go test ./...
go vet ./...
node --check internal/web/static/app.js
node --test scripts/ui.test.cjs
```

- Run checks appropriate to the change. Frontend checks require Node.js but add
  no application runtime dependency. For documentation-only changes, check links,
  paths, and consistency without rerunning unrelated application tests.
- Use `httptest` for Mock behavior. Test generated YAML directly without mutating
  Gateway fixtures. Process-argument unit tests must not launch a real Gateway.
- For relevant Mock changes, smoke-test `/healthz`, a control-plane JSON response,
  a stream ending in `[DONE]`, scenario priority/fallback, and unknown-route HTTP 501.
- For UI changes, verify scene selection/save, SSE editing, custom popup keyboard
  behavior, code/log readability, and narrow-screen overflow. Capture an actual
  browser screenshot; do not present a generated design image as an implemented UI.
- Use temporary profiles and separate loopback ports for verification. Do not
  overwrite the active user's profile or stop pre-existing services. Clean up
  only the test processes you started.
- Treat captured bodies, headers/API keys, profiles, and Gateway logs as sensitive
  local debug data. Do not commit real credentials or captured session data.

## 7. Documentation maintenance

Update this guide and `AGENTS.zh-CN.md` together when boundaries, configuration,
matching behavior, frontend conventions, or validation commands change. Keep
`README.md` focused on installation, use, and troubleshooting; these guides cover
maintenance and collaboration rules. Separate confirmed behavior from proposals.
