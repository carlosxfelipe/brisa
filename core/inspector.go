package core

import "fmt"

// RenderInspectorHTML returns the self-contained Brisa IPC Inspector HTML interface
// styled after GNOME 50 / GTK4 / Libadwaita (D-Spy / GNOME Builder style).
func (a *App) RenderInspectorHTML() string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en" translate="no">
<head>
  <meta charset="UTF-8" />
  <meta name="google" content="notranslate" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>%s — Brisa IPC Inspector</title>
  <link rel="stylesheet" href="/brisa/adwaita.css" />
  <script src="/brisa/sdk.js"></script>
  <style>
    :root {
      --divider-color: var(--row-divider-color);
      --dim-label: var(--dim-label-color);
      --button-hover: var(--button-hover-bg);
      --accent-bg: var(--accent-bg-color);
      --accent-hover: var(--accent-hover-bg);
      --danger-color: var(--destructive-color);
    }
    * { box-sizing: border-box; margin: 0; padding: 0; }
    body {
      font-family: var(--font-sans, "Cantarell", "Inter", system-ui, -apple-system, sans-serif);
      background: var(--window-bg-color);
      color: var(--window-fg-color);
      display: grid;
      grid-template-columns: 280px 1fr 330px;
      grid-template-rows: 46px 1fr;
      height: 100vh;
      overflow: hidden;
      font-size: 13.5px;
    }
    header {
      grid-column: 1 / -1;
      background: var(--headerbar-bg-color);
      border-bottom: 1px solid var(--headerbar-border-color);
      display: flex;
      align-items: center;
      justify-content: space-between;
      padding: 0 14px;
    }
    .brand { display: flex; align-items: center; gap: 8px; font-weight: 700; font-size: 14px; }
    .badge {
      background: var(--button-bg);
      color: var(--window-fg-color);
      padding: 2px 9px;
      border-radius: 999px;
      font-size: 11.5px;
      font-weight: 600;
    }
    aside, main, .events-col {
      overflow-y: auto;
      height: calc(100vh - 46px);
    }
    aside {
      background: var(--sidebar-bg-color);
      border-right: 1px solid var(--sidebar-border-color);
      padding: 10px 8px;
    }
    .events-col {
      background: var(--sidebar-bg-color);
      border-left: 1px solid var(--sidebar-border-color);
      padding: 12px;
      display: flex;
      flex-direction: column;
      gap: 8px;
    }
    main { padding: 22px; }
    h2 {
      font-size: 12px;
      font-weight: 700;
      color: var(--dim-label);
      padding: 4px 8px 8px;
    }
    .cmd-item {
      padding: 8px 11px;
      border-radius: var(--control-radius, 8px);
      cursor: default;
      margin-bottom: 3px;
      transition: background 120ms ease;
    }
    .cmd-item:hover { background: var(--nav-hover-bg); }
    .cmd-item.active {
      background: var(--nav-active-bg);
    }
    .cmd-name { font-family: ui-monospace, SFMono-Regular, monospace; font-size: 13px; font-weight: 600; color: var(--accent-color); }
    .cmd-sum { font-size: 11.5px; color: var(--dim-label); margin-top: 2px; }
    .boxed-group {
      background: var(--card-bg-color);
      border-radius: var(--card-radius, 12px);
      box-shadow: var(--card-shadow);
      padding: 16px;
      margin-bottom: 18px;
    }
    textarea {
      width: 100%%;
      background: var(--view-bg-color);
      color: var(--window-fg-color);
      border: 1px solid var(--divider-color);
      border-radius: var(--control-radius, 8px);
      padding: 10px 12px;
      font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
      font-size: 13px;
      min-height: 120px;
      resize: vertical;
      outline: none;
    }
    textarea:focus {
      box-shadow: var(--focus-ring);
    }
    .gtk-btn {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      gap: 7px;
      background: var(--button-bg);
      color: var(--window-fg-color);
      border: none;
      border-radius: var(--control-radius, 9px);
      height: 34px;
      min-height: 34px;
      min-width: 34px;
      padding: 0 14px;
      font-family: inherit;
      font-weight: 600;
      font-size: 14px;
      line-height: 1;
      white-space: nowrap;
      cursor: default;
      text-decoration: none;
    }
    .gtk-btn:hover { background: var(--button-hover); }
    .gtk-btn.suggested {
      background: var(--accent-bg);
      color: var(--accent-fg-color);
    }
    .gtk-btn.suggested:hover { background: var(--accent-hover); }
    pre {
      background: var(--view-bg-color);
      border: 1px solid var(--divider-color);
      border-radius: var(--control-radius, 8px);
      padding: 12px;
      font-size: 12.5px;
      overflow-x: auto;
      color: var(--view-fg-color);
    }
    .event-log {
      flex: 1;
      overflow-y: auto;
      display: flex;
      flex-direction: column;
      gap: 6px;
    }
    .event-entry {
      background: var(--view-bg-color);
      border: 1px solid var(--divider-color);
      border-radius: var(--control-radius, 8px);
      padding: 8px 10px;
      font-size: 11.5px;
      font-family: ui-monospace, monospace;
    }
  </style>
</head>
<body>
  <header>
    <div class="brand">
      <span>%s</span>
      <span class="badge">v%s</span>
      <span class="badge">D-Spy / Brisa IPC Inspector</span>
    </div>
    <div style="display:flex;gap:6px;">
      <a href="/brisa/schema.json" target="_blank" class="gtk-btn">Schema JSON</a>
      <a href="/brisa/types.d.ts" target="_blank" class="gtk-btn">TypeScript (.d.ts)</a>
    </div>
  </header>
  <aside>
    <h2>Registered Commands</h2>
    <div id="cmd-list"></div>
  </aside>
  <main id="detail-pane">
    <div class="boxed-group">Select a command on the left to inspect its REPR contract and execute IPC calls.</div>
  </main>
  <section class="events-col">
    <h2>Live SSE Event Stream</h2>
    <div class="event-log" id="event-log"></div>
  </section>
  <script>
    const cmds = (window.brisa && window.brisa.schemas) || [];
    const listEl = document.getElementById("cmd-list");
    const detailEl = document.getElementById("detail-pane");
    const eventLogEl = document.getElementById("event-log");

    function buildExample(schema) {
      if (!schema || !schema.properties) return {};
      const out = {};
      for (const [k, v] of Object.entries(schema.properties)) {
        if (v.example !== undefined) {
          if (v.type === "number") out[k] = Number(v.example);
          else if (v.type === "boolean") out[k] = v.example === "true";
          else out[k] = v.example;
        } else if (v.type === "string") out[k] = "";
        else if (v.type === "number") out[k] = 0;
        else if (v.type === "boolean") out[k] = false;
      }
      return out;
    }

    function selectCmd(cmd, el) {
      document.querySelectorAll(".cmd-item").forEach(n => n.classList.remove("active"));
      if (el) el.classList.add("active");
      const sample = JSON.stringify(buildExample(cmd.input), null, 2);
      detailEl.innerHTML =
        '<div class="boxed-group">' +
          '<div class="cmd-name" style="font-size:16px;margin-bottom:4px;">' + cmd.name + '</div>' +
          '<p style="color:var(--dim-label);font-size:13px;margin-bottom:12px;">' + (cmd.summary || "No summary provided") + '</p>' +
          '<div style="font-size:12px;color:var(--dim-label);margin-bottom:6px;font-weight:600;">Request Payload (JSON):</div>' +
          '<textarea id="rpc-input">' + sample + '</textarea>' +
          '<div style="margin-top:12px;display:flex;align-items:center;gap:12px;">' +
            '<button class="gtk-btn suggested" id="run-btn">Invoke Command</button>' +
            '<span id="rpc-status" style="font-size:12.5px;font-weight:600;color:var(--dim-label);"></span>' +
          '</div>' +
        '</div>' +
        '<div class="boxed-group">' +
          '<div style="font-size:12.5px;font-weight:700;margin-bottom:8px;">IPC Response</div>' +
          '<pre id="rpc-output">// Click "Invoke Command" to test via Brisa Bridge</pre>' +
        '</div>' +
        '<div class="boxed-group">' +
          '<div style="font-size:12.5px;font-weight:700;margin-bottom:8px;">REPR Schema Contract</div>' +
          '<pre>' + JSON.stringify({ input: cmd.input, output: cmd.output }, null, 2) + '</pre>' +
        '</div>';

      document.getElementById("run-btn").onclick = async () => {
        const statusEl = document.getElementById("rpc-status");
        const outEl = document.getElementById("rpc-output");
        const t0 = performance.now();
        try {
          const payload = JSON.parse(document.getElementById("rpc-input").value || "{}");
          const res = await window.brisa.invoke(cmd.name, payload);
          const ms = (performance.now() - t0).toFixed(2);
          statusEl.style.color = "var(--success-color)";
          statusEl.textContent = "200 OK (" + ms + " ms)";
          outEl.textContent = JSON.stringify(res, null, 2);
        } catch (err) {
          const ms = (performance.now() - t0).toFixed(2);
          statusEl.style.color = "var(--danger-color)";
          statusEl.textContent = (err.status || 400) + " " + (err.code || "ERROR") + " (" + ms + " ms)";
          outEl.textContent = JSON.stringify({
            code: err.code,
            message: err.message,
            status: err.status,
            errors: err.errors
          }, null, 2);
        }
      };
    }

    cmds.forEach((cmd, idx) => {
      const div = document.createElement("div");
      div.className = "cmd-item";
      div.innerHTML = '<div class="cmd-name">' + cmd.name + '</div><div class="cmd-sum">' + (cmd.summary || "") + '</div>';
      div.onclick = () => selectCmd(cmd, div);
      listEl.appendChild(div);
      if (idx === 0) selectCmd(cmd, div);
    });

    if (window.brisa) {
      window.brisa.on("*", (evt) => {
        const item = document.createElement("div");
        item.className = "event-entry";
        item.textContent = "[" + new Date().toLocaleTimeString() + "] " + evt.event + ": " + JSON.stringify(evt.data);
        eventLogEl.prepend(item);
      });
    }
  </script>
</body>
</html>`, a.config.Title, a.config.Title, a.config.Version)
}
