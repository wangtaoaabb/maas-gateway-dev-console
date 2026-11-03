// Keep native select values as the form data source; render the popup ourselves
// so the open menu has the same appearance on macOS, Windows and Linux.
const selectControls = new Map();
let openSelectControl = null;

function closeSelect() {
  if (!openSelectControl) return;
  const control = openSelectControl;
  control.menu.hidden = true;
  control.trigger.setAttribute("aria-expanded", "false");
  control.trigger.removeAttribute("aria-activedescendant");
  control.wrapper.classList.remove("is-open");
  openSelectControl = null;
}

function positionSelect(control) {
  const rect = control.trigger.getBoundingClientRect();
  const width = Math.min(Math.max(rect.width, 220), window.innerWidth - 24);
  const below = window.innerHeight - rect.bottom - 20;
  const above = rect.top - 20;
  control.menu.style.width = `${width}px`;
  control.menu.style.left = `${Math.max(12, Math.min(rect.left, window.innerWidth - width - 12))}px`;
  control.menu.style.maxHeight = `${Math.max(96, Math.min(320, Math.max(below, above)))}px`;
  const height = control.menu.getBoundingClientRect().height;
  control.menu.style.top = `${below >= height || below >= above ? rect.bottom + 8 : Math.max(12, rect.top - height - 8)}px`;
}

function highlightSelect(control, index, scroll = true) {
  control.highlight = index;
  [...control.menu.children].forEach((option, position) => option.classList.toggle("highlighted", position === index));
  const active = control.menu.children[index];
  if (active) {
    control.trigger.setAttribute("aria-activedescendant", active.id);
    if (scroll) {
      const top = active.offsetTop;
      const bottom = top + active.offsetHeight;
      if (top < control.menu.scrollTop) control.menu.scrollTop = top;
      else if (bottom > control.menu.scrollTop + control.menu.clientHeight) control.menu.scrollTop = bottom - control.menu.clientHeight;
    }
  }
}

function chooseSelect(control, index) {
  const option = control.select.options[index];
  if (!option || option.disabled) return;
  const changed = control.select.value !== option.value;
  control.select.value = option.value;
  closeSelect();
  if (changed) control.select.dispatchEvent(new Event("change", { bubbles: true }));
  syncSelect(control.select);
  control.trigger.focus({ preventScroll: true });
}

function renderSelectOptions(control) {
  control.menu.replaceChildren();
  [...control.select.options].forEach((option, index) => {
    const button = document.createElement("button");
    button.type = "button";
    button.className = "select-option";
    button.id = `${control.menu.id}-${index}`;
    button.setAttribute("role", "option");
    button.setAttribute("aria-selected", String(option.selected));
    button.tabIndex = -1;
    button.disabled = option.disabled;
    const text = document.createElement("span");
    text.textContent = option.textContent;
    const check = document.createElement("span");
    check.className = "select-check";
    check.setAttribute("aria-hidden", "true");
    check.textContent = option.selected ? "✓" : "";
    button.append(text, check);
    button.onclick = () => chooseSelect(control, index);
    button.onpointermove = () => { if (!option.disabled) highlightSelect(control, index, false); };
    control.menu.appendChild(button);
  });
}

function syncSelect(select) {
  const control = selectControls.get(select);
  if (!control) return;
  control.value.textContent = select.options[select.selectedIndex]?.textContent || "选择场景";
  control.trigger.disabled = select.disabled || !select.options.length;
  if (openSelectControl === control) {
    renderSelectOptions(control);
    positionSelect(control);
    highlightSelect(control, select.selectedIndex);
  }
}

function openSelect(control) {
  closeSelect();
  if (control.trigger.disabled) return;
  openSelectControl = control;
  renderSelectOptions(control);
  control.menu.hidden = false;
  control.wrapper.classList.add("is-open");
  control.trigger.setAttribute("aria-expanded", "true");
  positionSelect(control);
  highlightSelect(control, Math.max(0, control.select.selectedIndex));
}

function selectKeydown(control, event) {
  const open = openSelectControl === control;
  if (event.key === "Tab") { closeSelect(); return; }
  if (event.key === "Escape") { if (open) { event.preventDefault(); closeSelect(); } return; }
  if (["Enter", " "].includes(event.key)) {
    event.preventDefault();
    if (open) chooseSelect(control, control.highlight);
    else openSelect(control);
    return;
  }
  if (["ArrowDown", "ArrowUp", "Home", "End"].includes(event.key)) {
    event.preventDefault();
    if (!open) openSelect(control);
    const enabled = [...control.select.options].map((option, index) => option.disabled ? -1 : index).filter((index) => index >= 0);
    if (!enabled.length) return;
    const current = enabled.indexOf(control.highlight);
    let next = current;
    if (event.key === "Home") next = 0;
    else if (event.key === "End") next = enabled.length - 1;
    else if (open) next = (current + (event.key === "ArrowDown" ? 1 : -1) + enabled.length) % enabled.length;
    highlightSelect(control, enabled[Math.max(0, next)]);
    return;
  }
  if (event.key.length === 1 && !event.ctrlKey && !event.metaKey && !event.altKey) {
    event.preventDefault();
    const now = Date.now();
    control.query = (now - control.queryTime < 700 ? control.query : "") + event.key.toLowerCase();
    control.queryTime = now;
    if (!open) openSelect(control);
    const index = [...control.select.options].findIndex((option) => !option.disabled && option.textContent.toLowerCase().startsWith(control.query));
    if (index >= 0) highlightSelect(control, index);
  }
}

function enhanceSelects() {
  $$("select").forEach((select) => {
    if (selectControls.has(select)) return;
    const label = select.labels[0]?.firstChild?.textContent.trim() || select.id;
    const wrapper = document.createElement("span");
    wrapper.className = "custom-select";
    const trigger = document.createElement("button");
    trigger.type = "button";
    trigger.className = "select-trigger";
    trigger.setAttribute("role", "combobox");
    trigger.setAttribute("aria-label", label);
    trigger.setAttribute("aria-haspopup", "listbox");
    trigger.setAttribute("aria-expanded", "false");
    const value = document.createElement("span");
    value.className = "select-value";
    const arrow = document.createElement("span");
    arrow.className = "select-chevron";
    arrow.setAttribute("aria-hidden", "true");
    trigger.append(value, arrow);
    const menu = document.createElement("div");
    menu.id = `${select.id}-listbox`;
    menu.className = "select-menu";
    menu.hidden = true;
    menu.setAttribute("role", "listbox");
    menu.setAttribute("aria-label", label);
    trigger.setAttribute("aria-controls", menu.id);
    select.before(wrapper);
    wrapper.append(select, trigger);
    select.hidden = true;
    select.tabIndex = -1;
    select.setAttribute("aria-hidden", "true");
    document.body.appendChild(menu);
    const control = { select, wrapper, trigger, value, menu, highlight: 0, query: "", queryTime: 0 };
    selectControls.set(select, control);
    trigger.onclick = (event) => { event.preventDefault(); if (openSelectControl === control) closeSelect(); else openSelect(control); };
    trigger.onkeydown = (event) => selectKeydown(control, event);
    menu.onpointerdown = (event) => event.preventDefault();
    syncSelect(select);
  });
  document.addEventListener("pointerdown", (event) => {
    if (openSelectControl && !openSelectControl.wrapper.contains(event.target) && !openSelectControl.menu.contains(event.target)) closeSelect();
  });
  document.addEventListener("scroll", (event) => {
    if (openSelectControl && !openSelectControl.menu.contains(event.target)) closeSelect();
  }, true);
  window.addEventListener("resize", closeSelect);
}

const state = { profile: null, selectedRoute: null, routeDirty: false, profileDirty: false, sseEvents: [], responseMode: "body", requests: [], lastLogID: 0 };

const $ = (selector) => document.querySelector(selector);
const $$ = (selector) => [...document.querySelectorAll(selector)];

async function api(path, options = {}) {
  const response = await fetch(path, { headers: { "Content-Type": "application/json" }, ...options });
  const payload = await response.json().catch(() => ({}));
  if (!response.ok) throw new Error(payload.error || `${response.status} ${response.statusText}`);
  return payload;
}

function toast(message, error = false) {
  const node = $("#toast");
  node.textContent = message;
  node.className = `toast${error ? " error" : ""}`;
  setTimeout(() => node.classList.add("hidden"), 3500);
}

function valueAt(object, path) {
  return path.split(".").reduce((value, key) => value?.[key], object);
}

function setAt(object, path, value) {
  const parts = path.split(".");
  const leaf = parts.pop();
  const parent = parts.reduce((current, key) => current[key], object);
  parent[leaf] = value;
}

async function loadProfile() {
  state.profile = await api("/api/profile");
  $("#profile-name").textContent = state.profile.name;
  $$('[data-profile]').forEach((input) => {
    const value = valueAt(state.profile, input.dataset.profile);
    if (input.type === "checkbox") input.checked = Boolean(value);
    else input.value = value ?? "";
  });
  $("#profile-json").value = JSON.stringify(state.profile, null, 2);
  state.routeDirty = false;
  state.profileDirty = false;
  $("#route-unsaved").classList.add("hidden");
  renderRoutes();
  showRoute(state.selectedRoute);
}

function collectSettings() {
  $$('[data-profile]').forEach((input) => {
    let value = input.type === "checkbox" ? input.checked : input.value;
    if (input.type === "number") value = Number(value);
    setAt(state.profile, input.dataset.profile, value);
  });
}

async function saveProfile(message = "Profile saved") {
  stashRouteDraft();
  const result = await api("/api/profile", { method: "PUT", body: JSON.stringify(state.profile) });
  state.profile = result.profile;
  state.profileDirty = false;
  $("#route-unsaved").classList.add("hidden");
  $("#profile-json").value = JSON.stringify(state.profile, null, 2);
  toast(result.listener_restart_required ? `${message}; restart console to apply listener changes` : message);
  renderRoutes();
}

function interfaceKey(route) {
  return JSON.stringify([route.method.toUpperCase(), route.path]);
}

function routeGroups() {
  const groups = new Map();
  for (const route of state.profile.mock.routes) {
    const key = interfaceKey(route);
    if (!groups.has(key)) groups.set(key, []);
    groups.get(key).push(route);
  }
  return groups;
}

function selectedRoute() {
  return state.profile.mock.routes.find((route) => route.id === state.selectedRoute);
}

function markProfileDirty() {
  state.profileDirty = true;
  $("#route-unsaved").classList.remove("hidden");
}

function markRouteDirty() {
  state.routeDirty = true;
  markProfileDirty();
}

function renderScenarios() {
  const select = $("#route-scenario");
  select.replaceChildren();
  const current = selectedRoute();
  if (!current) return;
  const group = routeGroups().get(interfaceKey(current));
  [...group].sort((a, b) => b.priority - a.priority).forEach((route) => {
    const option = document.createElement("option");
    option.value = route.id;
    const format = route.response.sse?.length ? "SSE" : "普通响应";
    option.textContent = `${route.name || route.id} · Priority ${route.priority || 0} · ${format}${route.enabled ? "" : " · 已禁用"}`;
    select.appendChild(option);
  });
  select.value = current.id;
  syncSelect(select);
}

function renderRoutes() {
  const list = $("#route-list");
  list.replaceChildren();
  if (!selectedRoute()) state.selectedRoute = state.profile.mock.routes[0]?.id || null;
  const current = selectedRoute();
  $("#route-form").classList.toggle("hidden", !current);
  $("#route-empty").classList.toggle("hidden", Boolean(current));
  $("#save-route").disabled = !current;
  $("#delete-route").disabled = !current;
  if (!current) $("#route-title").textContent = "选择一个接口";
  for (const [key, group] of routeGroups()) {
    const first = group[0];
    const button = document.createElement("button");
    const active = current && interfaceKey(current) === key;
    button.className = `route-item${group.some((route) => route.enabled) ? "" : " off"}${active ? " active" : ""}`;
    const heading = document.createElement("div");
    heading.className = "route-item-heading";
    const name = document.createElement("strong");
    name.textContent = first.name || first.id;
    const count = document.createElement("span");
    count.className = "scenario-count";
    count.textContent = `${group.length} 个场景`;
    heading.append(name, count);
    const meta = document.createElement("div");
    meta.className = "route-item-meta";
    const method = document.createElement("span");
    method.className = "method-badge";
    method.textContent = first.method;
    const path = document.createElement("span");
    path.className = "route-path";
    path.textContent = first.path;
    meta.append(method, path);
    button.append(heading, meta);
    const description = group.find((route) => route.description)?.description;
    if (description) {
      const text = document.createElement("span");
      text.className = "route-description";
      text.textContent = description;
      button.appendChild(text);
    }
    button.onclick = () => {
      try { selectRoute(active ? current.id : first.id); }
      catch (error) { toast(error.message, true); }
    };
    list.appendChild(button);
  }
  renderScenarios();
  for (const select of selectControls.keys()) syncSelect(select);
}

function showRoute(id) {
  state.selectedRoute = id;
  const route = selectedRoute();
  state.routeDirty = false;
  if (!route) { renderRoutes(); return; }
  $("#route-title").textContent = `${route.method} ${route.path}`;
  $("#route-id").value = route.id;
  $("#route-name").value = route.name || "";
  const group = routeGroups().get(interfaceKey(route));
  $("#route-description").value = group.find((item) => item.description)?.description || "";
  $("#route-method").value = route.method;
  $("#route-path").value = route.path;
  $("#route-priority").value = route.priority || 0;
  $("#route-enabled").checked = route.enabled;
  $("#route-status").value = route.response.status;
  $("#route-delay").value = route.response.delay_ms || 0;
  $("#route-close").value = route.response.close_after_events || 0;
  $("#route-match").value = JSON.stringify(route.match || {}, null, 2);
  $("#route-headers").value = JSON.stringify(route.response.headers || {}, null, 2);
  $("#route-body").value = route.response.body || "";
  state.sseEvents = (route.response.sse || []).map((event) => ({ ...event }));
  $("#route-response-mode").value = state.sseEvents.length ? "sse" : "body";
  updateResponseMode();
  renderSSEEvents();
  renderRoutes();
}

function selectRoute(id) {
  const previousID = state.selectedRoute;
  stashRouteDraft();
  showRoute(state.profile.mock.routes.some((route) => route.id === id) ? id : state.selectedRoute || previousID);
}

function parseObject(value, label) {
  let result;
  try { result = JSON.parse(value || "{}"); }
  catch { throw new Error(`${label} 必须是有效的 JSON`); }
  if (!result || typeof result !== "object" || Array.isArray(result)) throw new Error(`${label} 必须是 JSON 对象`);
  return result;
}

function integerValue(value, label, min = 0, max = Number.MAX_SAFE_INTEGER) {
  const number = Number(value);
  if (String(value).trim() === "" || !Number.isSafeInteger(number) || number < min || number > max) {
    throw new Error(`${label} 必须是 ${min} 到 ${max} 之间的整数`);
  }
  return number;
}

function readRouteEditor() {
  const match = parseObject($("#route-match").value, "Request match");
  for (const key of ["headers", "query", "json"]) {
    if (match[key] !== undefined) parseObject(JSON.stringify(match[key]), `Request match.${key}`);
  }
  for (const key of ["headers", "query"]) {
    if (Object.values(match[key] || {}).some((value) => typeof value !== "string")) throw new Error(`Request match.${key} 的值必须是字符串`);
  }
  const headers = parseObject($("#route-headers").value, "Response headers");
  if (Object.values(headers).some((value) => typeof value !== "string")) throw new Error("Response headers 的值必须是字符串");
  const streaming = $("#route-response-mode").value === "sse";
  if (streaming && !state.sseEvents.length) throw new Error("SSE 响应至少需要一个事件");
  return {
    id: $("#route-id").value.trim(),
    name: $("#route-name").value.trim(),
    description: $("#route-description").value.trim(),
    enabled: $("#route-enabled").checked,
    priority: integerValue($("#route-priority").value, "Priority", Number.MIN_SAFE_INTEGER),
    method: $("#route-method").value,
    path: $("#route-path").value.trim(),
    match,
    response: {
      status: integerValue($("#route-status").value, "Status", 100, 599),
      headers,
      body: $("#route-body").value,
      sse: streaming ? state.sseEvents.map((event, index) => ({ data: event.data, delay_ms: integerValue(event.delay_ms ?? 0, `事件 ${index + 1} 的延迟`) })) : [],
      delay_ms: integerValue($("#route-delay").value, "Delay"),
      close_after_events: streaming ? integerValue($("#route-close").value, "提前断流事件数") : 0,
    },
  };
}

function stashRouteDraft() {
  if (!state.routeDirty || !selectedRoute()) return;
  const route = readRouteEditor();
  if (!route.id) throw new Error("场景 ID 不能为空");
  if (!route.path.startsWith("/")) throw new Error("Path 必须以 / 开头");
  if (state.profile.mock.routes.some((item) => item.id === route.id && item.id !== state.selectedRoute)) throw new Error("场景 ID 已存在，请使用不同的 ID");
  const index = state.profile.mock.routes.findIndex((item) => item.id === state.selectedRoute);
  state.profile.mock.routes[index] = route;
  for (const sibling of state.profile.mock.routes) {
    if (interfaceKey(sibling) === interfaceKey(route)) sibling.description = route.description;
  }
  state.selectedRoute = route.id;
  state.routeDirty = false;
}

async function saveRoute() {
  if (!selectedRoute()) return;
  markRouteDirty();
  stashRouteDraft();
  await saveProfile("场景已保存，后续 Mock 请求立即生效");
  showRoute(state.selectedRoute);
}

function newRouteID() {
  return `route-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
}

function addRoute() {
  stashRouteDraft();
  const id = newRouteID();
  let path = "/api/example";
  let suffix = 1;
  while (state.profile.mock.routes.some((route) => route.method === "POST" && route.path === path)) path = `/api/example-${suffix++}`;
  state.profile.mock.routes.push({
    id, name: "新接口默认场景", description: "", enabled: true, priority: 0, method: "POST", path,
    match: {}, response: { status: 200, headers: { "Content-Type": "application/json" }, body: "{}", sse: [], delay_ms: 0, close_after_events: 0 },
  });
  markProfileDirty();
  showRoute(id);
}

function addScenario(duplicate = false) {
  stashRouteDraft();
  const source = selectedRoute();
  if (!source) return;
  const route = duplicate ? JSON.parse(JSON.stringify(source)) : {
    method: source.method, path: source.path, description: source.description,
    match: {}, response: { status: 200, headers: { "Content-Type": "application/json" }, body: "{}", sse: [], delay_ms: 0, close_after_events: 0 },
  };
  route.id = newRouteID();
  route.name = duplicate ? `${source.name || source.id}（副本）` : "新匹配场景";
  route.enabled = false;
  route.priority = source.priority || 0;
  state.profile.mock.routes.push(route);
  markProfileDirty();
  showRoute(route.id);
  toast("新场景默认禁用；配置匹配条件后勾选 Enabled 并保存");
}

async function deleteRoute() {
  const current = selectedRoute();
  if (!current) return;
  const sibling = state.profile.mock.routes.find((route) => route.id !== current.id && interfaceKey(route) === interfaceKey(current));
  state.routeDirty = false;
  state.profile.mock.routes = state.profile.mock.routes.filter((route) => route.id !== current.id);
  markProfileDirty();
  showRoute(sibling?.id || state.profile.mock.routes[0]?.id || null);
  await saveProfile("场景已删除");
}

function updateResponseMode(updateHeaders = false) {
  const streaming = $("#route-response-mode").value === "sse";
  if (updateHeaders) {
    const headers = parseObject($("#route-headers").value, "Response headers");
    const contentTypeKey = Object.keys(headers).find((key) => key.toLowerCase() === "content-type");
    if (streaming || !contentTypeKey || String(headers[contentTypeKey]).startsWith("text/event-stream")) {
      if (contentTypeKey) delete headers[contentTypeKey];
      headers["Content-Type"] = streaming ? "text/event-stream" : "application/json";
    }
    $("#route-headers").value = JSON.stringify(headers, null, 2);
  }
  $("#body-editor").classList.toggle("hidden", streaming);
  $("#sse-editor").classList.toggle("hidden", !streaming);
  state.responseMode = streaming ? "sse" : "body";
}

function displayEventData(data) {
  try { return JSON.stringify(JSON.parse(data), null, 2); }
  catch { return data; }
}

function eventButton(text, action, disabled = false) {
  const button = document.createElement("button");
  button.textContent = text;
  button.className = "secondary compact";
  button.disabled = disabled;
  button.onclick = action;
  return button;
}

function moveSSEEvent(index, offset) {
  const target = index + offset;
  if (target < 0 || target >= state.sseEvents.length) return;
  [state.sseEvents[index], state.sseEvents[target]] = [state.sseEvents[target], state.sseEvents[index]];
  markRouteDirty();
  renderSSEEvents();
}

function addSSEEvent(data = "{}") {
  state.sseEvents.push({ data, delay_ms: 0 });
  markRouteDirty();
  renderSSEEvents();
}

function renderSSEEvents() {
  const list = $("#sse-events");
  list.replaceChildren();
  if (!state.sseEvents.length) {
    const empty = document.createElement("p");
    empty.className = "muted small";
    empty.textContent = "还没有事件。点击「+ 事件」开始配置，或添加 [DONE] 结束标记。";
    list.appendChild(empty);
  }
  state.sseEvents.forEach((event, index) => {
    const card = document.createElement("section");
    card.className = "sse-event";
    const heading = document.createElement("div");
    heading.className = "sse-event-heading";
    const title = document.createElement("strong");
    title.textContent = `事件 ${index + 1}${event.data === "[DONE]" ? " · 结束标记" : ""}`;
    const actions = document.createElement("div");
    actions.className = "inline-actions";
    const data = document.createElement("textarea");
    data.className = "code sse-event-data";
    data.setAttribute("aria-label", `事件 ${index + 1} data`);
    data.spellcheck = false;
    data.value = displayEventData(event.data);
    data.rows = Math.min(18, Math.max(4, data.value.split("\n").length));
    data.oninput = () => { event.data = data.value; markRouteDirty(); };
    actions.append(
      eventButton("格式化 JSON", () => {
        if (event.data === "[DONE]") return;
        try { event.data = JSON.stringify(JSON.parse(event.data), null, 2); data.value = event.data; data.rows = Math.min(18, Math.max(4, data.value.split("\n").length)); markRouteDirty(); }
        catch { toast(`事件 ${index + 1} 的 data 不是 JSON；普通文本无需格式化`, true); }
      }),
      eventButton("上移", () => moveSSEEvent(index, -1), index === 0),
      eventButton("下移", () => moveSSEEvent(index, 1), index === state.sseEvents.length - 1),
      eventButton("删除", () => { state.sseEvents.splice(index, 1); markRouteDirty(); renderSSEEvents(); }),
    );
    heading.append(title, actions);
    const delayLabel = document.createElement("label");
    delayLabel.className = "sse-event-delay";
    delayLabel.textContent = "发送前延迟（ms）";
    const delay = document.createElement("input");
    delay.type = "number";
    delay.min = "0";
    delay.step = "1";
    delay.value = event.delay_ms || 0;
    delay.oninput = () => { event.delay_ms = delay.value; markRouteDirty(); };
    delayLabel.appendChild(delay);
    card.append(heading, delayLabel, data);
    list.appendChild(card);
  });
}

async function refreshRuntime() {
  try {
    const runtime = await api("/api/runtime");
    const gateway = runtime.gateway;
    $("#mock-status").textContent = `Mock · ${runtime.mock.status}`;
    $("#mock-status").className = "status-chip good";
    $("#gateway-status").textContent = `Gateway · ${gateway.status}`;
    $("#gateway-status").className = `status-chip ${gateway.status === "running" ? "good" : gateway.status === "failed" ? "bad" : "warn"}`;
    const details = [`Mock ${runtime.mock.url}`];
    if (gateway.gateway_url) details.push(`Gateway ${gateway.gateway_url}`);
    if (gateway.mode === "debug") details.push(`Delve ${gateway.delve_address}`);
    if (gateway.error) details.push(`Error: ${gateway.error}`);
    $("#runtime-detail").textContent = details.join(" · ");
  } catch (error) {
    $("#mock-status").textContent = "Mock · unavailable";
    $("#mock-status").className = "status-chip bad";
  }
}

async function gatewayAction(action, mode) {
  collectSettings();
  await saveProfile("Profile saved");
  await api(`/api/gateway/${action}`, { method: "POST", body: JSON.stringify({ mode }) });
  toast(`Gateway ${action} requested`);
  await refreshRuntime();
}

async function refreshRequests() {
  const payload = await api("/api/requests");
  state.requests = payload.requests || [];
  const rows = $("#request-rows");
  rows.innerHTML = "";
  let unmatched = 0;
  [...state.requests].reverse().forEach((request) => {
    if (!request.matched) unmatched++;
    const row = document.createElement("tr");
    if (!request.matched) row.className = "unmatched";
    [new Date(request.time).toLocaleTimeString(), request.method, request.path, request.route_id || "UNCONFIGURED", request.status, `${request.duration_ms} ms`].forEach((value) => {
      const cell = document.createElement("td"); cell.textContent = value; row.appendChild(cell);
    });
    row.onclick = () => { $("#request-detail").textContent = JSON.stringify(request, null, 2); };
    rows.appendChild(row);
  });
  $("#unmatched-count").textContent = unmatched;
}

function stripLogANSI(line) {
  return line.replace(/\x1b\][^\x07\x1b]*(?:\x07|\x1b\\)/g, "").replace(/\x1b\[[0-?]*[ -/]*[@-~]/g, "");
}

function logLevel(line) {
  let level = "";
  try {
    const record = JSON.parse(line);
    level = String(record?.level || record?.severity || "").toLowerCase();
  } catch { /* Plain-text logs are also supported. */ }
  if (!level) {
    const explicit = line.match(/\b(?:level|severity)["']?\s*[:=]\s*["']?(trace|debug|info|warn(?:ing)?|error|fatal|panic)\b/i);
    const token = line.match(/\b(trace|debug|info|warn(?:ing)?|error|fatal|panic)\b/i);
    level = (explicit?.[1] || token?.[1] || "").toLowerCase();
  }
  if (["error", "fatal", "panic"].includes(level)) return "error";
  if (["warn", "warning"].includes(level)) return "warn";
  if (level === "info") return "info";
  if (["debug", "trace"].includes(level)) return "debug";
  return "plain";
}

function appendLog(entry) {
  const line = stripLogANSI(entry.line);
  const row = document.createElement("span");
  row.className = `log-line log-${logLevel(line)}`;
  const time = document.createElement("span");
  time.className = "log-time";
  time.textContent = `${new Date(entry.time).toLocaleTimeString()} `;
  const source = document.createElement("span");
  source.className = "log-source";
  source.textContent = `[${entry.source}] `;
  const message = document.createElement("span");
  message.textContent = line;
  row.append(time, source, message, document.createTextNode("\n"));
  $("#log-lines").appendChild(row);
}

async function refreshLogs() {
  const payload = await api(`/api/logs?after=${state.lastLogID}`);
  const lines = $("#log-lines");
  const follow = lines.scrollHeight - lines.scrollTop - lines.clientHeight < 50;
  for (const entry of payload.logs || []) {
    state.lastLogID = Math.max(state.lastLogID, entry.id);
    appendLog(entry);
  }
  while (lines.children.length > 5000) lines.firstElementChild.remove();
  if (follow && (payload.logs || []).length) lines.scrollTop = lines.scrollHeight;
}

function bindEvents() {
  $$(".tab").forEach((tab) => tab.onclick = () => {
    closeSelect();
    $$(".tab").forEach((node) => node.classList.toggle("active", node === tab));
    $$(".tab-page").forEach((page) => page.classList.toggle("active", page.id === tab.dataset.tab));
  });
  $("#save-settings").onclick = async () => { try { collectSettings(); await saveProfile(); } catch (error) { toast(error.message, true); } };
  $("#save-profile-json").onclick = async () => { try { const profile = JSON.parse($("#profile-json").value); state.routeDirty = false; state.profile = profile; await saveProfile(); await loadProfile(); } catch (error) { toast(error.message, true); } };
  $("#save-route").onclick = async () => { try { await saveRoute(); } catch (error) { toast(error.message, true); } };
  $("#add-route").onclick = () => { try { addRoute(); } catch (error) { toast(error.message, true); } };
  $("#add-scenario").onclick = () => { try { addScenario(); } catch (error) { toast(error.message, true); } };
  $("#duplicate-scenario").onclick = () => { try { addScenario(true); } catch (error) { toast(error.message, true); } };
  $("#route-scenario").onchange = () => { try { selectRoute($("#route-scenario").value); } catch (error) { renderScenarios(); toast(error.message, true); } };
  const dirtyInput = (event) => { if (event.target.id !== "route-scenario") markRouteDirty(); };
  $("#route-form").addEventListener("input", dirtyInput);
  $("#route-form").addEventListener("change", dirtyInput);
  $("#route-response-mode").onchange = () => {
    try { updateResponseMode(true); markRouteDirty(); }
    catch (error) { $("#route-response-mode").value = state.responseMode; toast(error.message, true); }
  };
  $("#add-sse-event").onclick = () => addSSEEvent();
  $("#add-sse-done").onclick = () => addSSEEvent("[DONE]");
  $("#delete-route").onclick = async () => { try { await deleteRoute(); } catch (error) { toast(error.message, true); } };
  $("#start-run").onclick = async () => { try { await gatewayAction("start", "run"); } catch (error) { toast(error.message, true); } };
  $("#start-debug").onclick = async () => { try { await gatewayAction("start", "debug"); } catch (error) { toast(error.message, true); } };
  $("#restart").onclick = async () => { try { const mode = (await api("/api/runtime")).gateway.mode || "run"; await gatewayAction("restart", mode); } catch (error) { toast(error.message, true); } };
  $("#stop").onclick = async () => { try { await api("/api/gateway/stop", { method: "POST", body: "{}" }); await refreshRuntime(); } catch (error) { toast(error.message, true); } };
  $("#clear-requests").onclick = async () => { await api("/api/requests", { method: "DELETE" }); await refreshRequests(); };
  $("#clear-logs").onclick = async () => { await api("/api/logs", { method: "DELETE" }); state.lastLogID = 0; $("#log-lines").textContent = ""; };
  window.addEventListener("beforeunload", (event) => {
    if (state.profileDirty) { event.preventDefault(); event.returnValue = ""; }
  });
}

async function main() {
  enhanceSelects();
  bindEvents();
  try {
    await loadProfile();
    await Promise.all([refreshRuntime(), refreshRequests(), refreshLogs()]);
  } catch (error) {
    toast(error.message, true);
  }
  setInterval(refreshRuntime, 1000);
  setInterval(refreshRequests, 1200);
  setInterval(refreshLogs, 700);
}

main();
