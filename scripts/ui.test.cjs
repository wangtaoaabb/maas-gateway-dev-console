const { test } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');

const root = path.resolve(__dirname, '..');
const source = fs.readFileSync(path.join(root, 'internal/web/static/app.js'), 'utf8').replace(/main\(\);\s*$/, '');
const fixture = JSON.parse(fs.readFileSync(path.join(root, 'profiles/default.json'), 'utf8'));

// Only model the DOM operations needed by the editor and plain-text log renderer.
// Real browser saves and HTTP matching are verified separately against a temporary profile.
function harness() {
  class Element {
    constructor() {
      this.children = [];
      this.value = '';
      this.checked = false;
      this.classes = new Set();
      this.classList = {
        add: (name) => this.classes.add(name),
        remove: (name) => this.classes.delete(name),
        toggle: (name, on) => on ? this.classes.add(name) : this.classes.delete(name),
      };
    }
    append(...children) { this.children.push(...children); }
    appendChild(child) { this.children.push(child); }
    replaceChildren(...children) { this.children = children; }
    setAttribute(name, value) { this[name] = value; }
    set innerHTML(value) { throw new Error(`Unexpected HTML rendering: ${value}`); }
  }
  const nodes = new Map();
  const document = {
    querySelector(selector) {
      if (!nodes.has(selector)) nodes.set(selector, new Element());
      return nodes.get(selector);
    },
    querySelectorAll: () => [],
    createElement: () => new Element(),
    createTextNode: (text) => ({ textContent: text }),
  };
  const context = vm.createContext({ document, setTimeout: () => 0 });
  vm.runInContext(source, context);
  const run = (code) => vm.runInContext(code, context);
  run(`state.profile = ${JSON.stringify(fixture)}; renderRoutes(); showRoute('chat-stream');`);
  return { run, node: (selector) => document.querySelector(selector) };
}

function plain(value) { return JSON.parse(JSON.stringify(value)); }

test('existing profiles group same method and path without changing matching order', () => {
  const { run, node } = harness();
  const expected = new Set(fixture.mock.routes.map((r) => JSON.stringify([r.method.toUpperCase(), r.path]))).size;
  assert.equal(node('#route-list').children.length, expected);
  assert.equal(node('#route-scenario').children.length, 2);
  assert.deepEqual(plain(run('state.profile.mock.routes.map(route => route.id)')), fixture.mock.routes.map((r) => r.id));
});

test('duplicating an SSE scenario creates an independent, disabled branch', () => {
  const { run } = harness();
  const before = plain(run('selectedRoute()'));
  run('addScenario(true)');
  const duplicate = plain(run('selectedRoute()'));
  assert.notEqual(duplicate.id, before.id);
  assert.equal(duplicate.enabled, false);
  assert.deepEqual(duplicate.response.sse, before.response.sse);
  run('selectedRoute().response.sse[0].data = "changed copy"');
  assert.deepEqual(plain(run('state.profile.mock.routes.find(route => route.id === "chat-stream")')), before);
});

test('valid edits and SSE data survive switching to another scenario and back', () => {
  const { run, node } = harness();
  node('#route-name').value = 'edited stream';
  node('#route-description').value = '同接口说明';
  run('state.sseEvents[0].data = "raw text"; state.sseEvents[0].delay_ms = 15; markRouteDirty(); selectRoute("chat-json"); selectRoute("chat-stream")');
  assert.equal(node('#route-name').value, 'edited stream');
  assert.equal(node('#route-description').value, '同接口说明');
  assert.equal(run('state.sseEvents[0].data'), 'raw text');
  assert.equal(run('state.sseEvents[0].delay_ms'), 15);
  assert.equal(run('state.profile.mock.routes.find(route => route.id === "chat-json").description'), '同接口说明');
  assert.equal(run('state.profileDirty'), true);
});

test('invalid match JSON and duplicate IDs keep the current draft and original profile intact', () => {
  const { run, node } = harness();
  node('#route-match').value = '{';
  run('markRouteDirty()');
  assert.throws(() => run('selectRoute("chat-json")'), /Request match/);
  assert.equal(run('state.selectedRoute'), 'chat-stream');
  assert.equal(node('#route-match').value, '{');
  node('#route-match').value = '{}';
  node('#route-id').value = 'chat-json';
  assert.throws(() => run('stashRouteDraft()'), /ID 已存在/);
  assert.equal(run('state.profile.mock.routes.filter(route => route.id === "chat-json").length'), 1);
});

test('SSE requires events and nonnegative integer delays; ordinary responses omit SSE', () => {
  const { run, node } = harness();
  run('state.sseEvents[0].delay_ms = -1');
  assert.throws(() => run('readRouteEditor()'), /延迟/);
  run('state.sseEvents = []');
  assert.throws(() => run('readRouteEditor()'), /至少需要一个事件/);
  node('#route-response-mode').value = 'body';
  assert.deepEqual(plain(run('readRouteEditor().response.sse')), []);
  assert.equal(run('readRouteEditor().response.close_after_events'), 0);
});

test('response mode updates Content-Type including lowercase existing header keys', () => {
  const { run, node } = harness();
  node('#route-headers').value = '{"content-type":"application/json","X-Test":"keep"}';
  node('#route-response-mode').value = 'sse';
  run('updateResponseMode(true)');
  assert.deepEqual(JSON.parse(node('#route-headers').value), { 'Content-Type': 'text/event-stream', 'X-Test': 'keep' });
  node('#route-response-mode').value = 'body';
  run('updateResponseMode(true)');
  assert.equal(JSON.parse(node('#route-headers').value)['Content-Type'], 'application/json');
});

test('JSON and text log levels use declared severity, not error words in the message', () => {
  const { run } = harness();
  const cases = [
    ['{"level":"info","msg":"previous error resolved"}', 'info'],
    ['{"severity":"WARNING","message":"retry"}', 'warn'],
    ['time=now level=ERROR msg="failed"', 'error'],
    ['[DEBUG] request details', 'debug'],
    ['2026-10-24T10:00:00Z INFO connected', 'info'],
    ['FATAL failed to start', 'error'],
    ['PANIC failed', 'error'],
    ['TRACE request', 'debug'],
    ['plain output', 'plain'],
  ];
  for (const [line, expected] of cases) assert.equal(run(`logLevel(${JSON.stringify(line)})`), expected, line);
});

test('ANSI sequences are removed and untrusted log markup remains plain text', () => {
  const { run, node } = harness();
  const payload = '\u001b[31mERROR <img src=x onerror=alert(1)>\u001b[0m';
  const clean = 'ERROR <img src=x onerror=alert(1)>';
  assert.equal(run(`stripLogANSI(${JSON.stringify(payload)})`), clean);
  assert.equal(run('stripLogANSI("\\x1b]0;title\\x07plain")'), 'plain');
  run(`appendLog({ line: ${JSON.stringify(payload)}, source: 'gateway', time: '2026-10-24T00:00:00Z' })`);
  const row = node('#log-lines').children[0];
  assert.equal(row.className, 'log-line log-error');
  assert.equal(row.children[2].textContent, clean);
  assert.equal(row.children[2].children.length, 0);
});

test('highlighting a popup option scrolls only the menu instead of the document', () => {
  const { run } = harness();
  run(`
    const popupOption = {
      id: 'popup-option', offsetTop: 200, offsetHeight: 40,
      classList: { toggle() {} },
      scrollIntoView() { throw new Error('must not scroll document'); },
    };
    const popupControl = {
      trigger: { setAttribute() {} },
      menu: { children: [popupOption], scrollTop: 0, clientHeight: 100 },
    };
    highlightSelect(popupControl, 0);
  `);
  assert.equal(run('popupControl.menu.scrollTop'), 140);
});
