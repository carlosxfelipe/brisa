package core

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// GenerateJSSDK generates the runtime JavaScript bridge served at /brisa/sdk.js.
func (a *App) GenerateJSSDK() string {
	spec := a.InspectorSpec()
	commandsJSON, _ := json.Marshal(spec.Commands)
	initialOSTheme := DetectSystemTheme()

	return fmt.Sprintf(`/* Brisa Desktop SDK - Auto-generated at runtime (Zero Dependencies) */
(function (global) {
  "use strict";

  const TOKEN = %q;
  const APP_ID = %q;
  const APP_TITLE = %q;
  const UI_THEME = %q;
  const INITIAL_OS_THEME = %q;
  const COMMAND_SCHEMAS = %s;

  const listeners = new Map();
  let sse = null;

  function syncOSTheme(theme) {
    if (!theme || typeof document === "undefined") return;
    if (document.documentElement) {
      document.documentElement.setAttribute("data-os-theme", theme);
      document.documentElement.setAttribute("data-ui-theme", UI_THEME);
      document.documentElement.style.colorScheme = theme;
    }
  }

  syncOSTheme(INITIAL_OS_THEME);

  if (typeof window !== "undefined" && typeof window.matchMedia === "function") {
    try {
      const mq = window.matchMedia("(prefers-color-scheme: dark)");
      if (typeof mq.addEventListener === "function") {
        mq.addEventListener("change", function (e) {
          syncOSTheme(e.matches ? "dark" : "light");
        });
      }
    } catch (_) {}
  }

  function connectSSE() {
    if (typeof EventSource === "undefined") return;
    const url = "/brisa/events?token=" + encodeURIComponent(TOKEN);
    sse = new EventSource(url);

    sse.onmessage = function (e) {
      try {
        const msg = JSON.parse(e.data);
        if (msg && msg.event) {
          dispatchLocal(msg.event, msg.data);
        }
      } catch (_) {}
    };

    sse.onerror = function () {
      if (sse) sse.close();
      setTimeout(connectSSE, 1500);
    };
  }

  function dispatchLocal(event, data) {
    if (event === "brisa:os:theme" && data && data.theme) {
      syncOSTheme(data.theme);
    } else if (event === "brisa:window:title" && data && data.title) {
      document.title = data.title;
    } else if (event === "brisa:window:size" && data && typeof window.resizeTo === "function") {
      window.resizeTo(data.width, data.height);
    } else if (event === "brisa:window:position" && data && typeof window.moveTo === "function") {
      window.moveTo(data.x, data.y);
    } else if (event === "brisa:window:close" && typeof window.close === "function") {
      window.close();
    } else if (event === "brisa:window:eval" && data && data.expression) {
      try { (0, eval)(data.expression); } catch (err) { console.error("[Brisa Eval]", err); }
    } else if (event === "brisa:clipboard:write" && data && typeof data.text === "string") {
      if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(data.text).catch(function () {});
      }
    }

    const set = listeners.get(event);
    if (set) {
      set.forEach(function (fn) {
        try { fn(data); } catch (err) { console.error("[Brisa Event Handler]", err); }
      });
    }
    const wildcard = listeners.get("*");
    if (wildcard) {
      wildcard.forEach(function (fn) {
        try { fn({ event: event, data: data }); } catch (err) { console.error("[Brisa Wildcard Handler]", err); }
      });
    }
  }

  class BrisaError extends Error {
    constructor(errObj) {
      super((errObj && errObj.message) || "Brisa IPC Error");
      this.name = "BrisaError";
      this.code = (errObj && errObj.code) || "BRISA_ERROR";
      this.status = (errObj && errObj.status) || 500;
      this.command = errObj && errObj.command;
      this.requestId = errObj && errObj.requestId;
      this.errors = (errObj && errObj.errors) || null;
      this.details = (errObj && errObj.details) || null;
    }
  }

  async function invoke(command, payload) {
    const res = await fetch("/brisa/rpc", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "X-Brisa-Token": TOKEN
      },
      body: JSON.stringify({
        command: command,
        payload: payload === undefined ? {} : payload
      })
    });

    const body = await res.json().catch(function () { return null; });
    if (!res.ok || (body && body.error)) {
      throw new BrisaError((body && body.error) || body || { message: res.statusText, status: res.status });
    }
    return body && body.data !== undefined ? body.data : body;
  }

  async function batch(calls) {
    const res = await fetch("/brisa/rpc/batch", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "X-Brisa-Token": TOKEN
      },
      body: JSON.stringify(calls)
    });
    const body = await res.json();
    if (!res.ok) {
      throw new BrisaError(body || { message: res.statusText, status: res.status });
    }
    return body;
  }

  function on(event, handler) {
    if (!listeners.has(event)) {
      listeners.set(event, new Set());
    }
    listeners.get(event).add(handler);
    return function unsubscribe() {
      const set = listeners.get(event);
      if (set) set.delete(handler);
    };
  }

  function once(event, handler) {
    const off = on(event, function (data) {
      off();
      handler(data);
    });
    return off;
  }

  async function emit(event, data) {
    return invoke("$brisa.events.emit", { event: event, data: data });
  }

  const call = {};
  const commands = {};

  COMMAND_SCHEMAS.forEach(function (cmd) {
    const fn = function (payload) {
      return invoke(cmd.name, payload);
    };
    commands[cmd.name] = fn;

    const parts = cmd.name.split(".");
    let cursor = call;
    for (let i = 0; i < parts.length - 1; i++) {
      const p = parts[i];
      if (!cursor[p] || typeof cursor[p] !== "object") {
        cursor[p] = {};
      }
      cursor = cursor[p];
    }
    cursor[parts[parts.length - 1]] = fn;
  });

  const brisa = {
    appId: APP_ID,
    title: APP_TITLE,
    theme: UI_THEME,
    schemas: COMMAND_SCHEMAS,
    BrisaError: BrisaError,
    invoke: invoke,
    batch: batch,
    call: call,
    commands: commands,
    on: on,
    once: once,
    emit: emit,

    window: {
      setTitle: function (title) { return invoke("$brisa.window.setTitle", { title: title }); },
      setSize: function (width, height) { return invoke("$brisa.window.setSize", { width: width, height: height }); },
      setMinSize: function (width, height) { return invoke("$brisa.window.setMinSize", { width: width, height: height }); },
      setPosition: function (x, y) { return invoke("$brisa.window.setPosition", { x: x, y: y }); },
      minimize: function () { return invoke("$brisa.window.minimize", {}); },
      maximize: function () { return invoke("$brisa.window.maximize", {}); },
      restore: function () { return invoke("$brisa.window.restore", {}); },
      setFullscreen: function (fullscreen) { return invoke("$brisa.window.setFullscreen", { fullscreen: !!fullscreen }); },
      close: function () { return invoke("$brisa.window.close", {}); },
      state: function () { return invoke("$brisa.window.state", {}); }
    },

    dialog: {
      info: function (title, message) { return invoke("$brisa.dialog.message", { title: title, message: message, kind: "info" }); },
      warn: function (title, message) { return invoke("$brisa.dialog.message", { title: title, message: message, kind: "warning" }); },
      error: function (title, message) { return invoke("$brisa.dialog.message", { title: title, message: message, kind: "error" }); },
      confirm: function (title, message) { return invoke("$brisa.dialog.confirm", { title: title, message: message }); },
      openFile: function (opts) { return invoke("$brisa.dialog.openFile", opts || {}); },
      openMultipleFiles: function (opts) { return invoke("$brisa.dialog.openMultipleFiles", opts || {}); },
      openDirectory: function (opts) { return invoke("$brisa.dialog.openDirectory", opts || {}); },
      saveFile: function (opts) { return invoke("$brisa.dialog.saveFile", opts || {}); }
    },

    clipboard: {
      writeText: function (text) { return invoke("$brisa.clipboard.writeText", { text: String(text) }); },
      readText: function () { return invoke("$brisa.clipboard.readText", {}); }
    },

    store: {
      get: function (key) { return invoke("$brisa.store.get", { key: key }); },
      set: function (key, value) { return invoke("$brisa.store.set", { key: key, value: value }); },
      delete: function (key) { return invoke("$brisa.store.delete", { key: key }); },
      keys: function () { return invoke("$brisa.store.keys", {}); },
      all: function () { return invoke("$brisa.store.all", {}); },
      clear: function () { return invoke("$brisa.store.clear", {}); }
    },

    notify: function (opts) { return invoke("$brisa.notify", opts || {}); },
    openExternal: function (target) { return invoke("$brisa.openExternal", { target: target }); }
  };

  function upgradeGtkDropdowns() {
    if (typeof document === "undefined") return;
    const selects = document.querySelectorAll("select.gtk-dropdown:not([data-adw-upgraded])");
    selects.forEach(function (sel) {
      sel.setAttribute("data-adw-upgraded", "1");
      sel.style.display = "none";

      const wrap = document.createElement("div");
      wrap.className = "gtk-dropdown-wrap";
      sel.parentNode.insertBefore(wrap, sel);
      wrap.appendChild(sel);

      const btn = document.createElement("button");
      btn.type = "button";
      btn.className = "gtk-dropdown-btn";
      const labelSpan = document.createElement("span");
      labelSpan.className = "gtk-dropdown-label";
      btn.appendChild(labelSpan);
      btn.insertAdjacentHTML(
        "beforeend",
        '<svg class="gtk-dropdown-chevron" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4 6l4 4 4-4"/></svg>'
      );
      wrap.appendChild(btn);

      const menu = document.createElement("div");
      menu.className = "gtk-popover-menu";
      menu.setAttribute("role", "listbox");
      document.body.appendChild(menu);

      function syncItems() {
        const opts = Array.from(sel.options);
        const activeOpt = opts.find(function (o) { return o.value === sel.value; }) || opts[0];
        labelSpan.textContent = activeOpt ? activeOpt.textContent : "";
        menu.innerHTML = "";
        opts.forEach(function (opt) {
          const item = document.createElement("button");
          item.type = "button";
          item.className = "gtk-popover-item" + (opt.value === sel.value ? " selected" : "");
          item.setAttribute("role", "option");
          const txt = document.createElement("span");
          txt.textContent = opt.textContent;
          item.appendChild(txt);
          item.insertAdjacentHTML(
            "beforeend",
            '<svg class="gtk-popover-check" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="2.1" stroke-linecap="round" stroke-linejoin="round"><polyline points="13.5 4.5 6.5 11.5 2.5 7.5"/></svg>'
          );
          item.addEventListener("click", function (ev) {
            ev.stopPropagation();
            sel.value = opt.value;
            syncItems();
            closeAllPopovers();
            sel.dispatchEvent(new Event("change", { bubbles: true }));
          });
          menu.appendChild(item);
        });
      }

      syncItems();
      sel.addEventListener("change", syncItems);

      btn.addEventListener("click", function (ev) {
        ev.stopPropagation();
        const isOpen = menu.classList.contains("open");
        closeAllPopovers();
        if (!isOpen) {
          syncItems();
          const rect = btn.getBoundingClientRect();
          menu.style.top = (rect.bottom + 6) + "px";
          menu.style.right = Math.max(8, window.innerWidth - rect.right) + "px";
          menu.style.minWidth = Math.max(148, rect.width) + "px";
          btn.classList.add("open");
          menu.classList.add("open");
        }
      });
    });
  }

  function closeAllPopovers() {
    if (typeof document === "undefined") return;
    document.querySelectorAll(".gtk-popover-menu.open, .gtk-dropdown-btn.open").forEach(function (el) {
      el.classList.remove("open");
    });
  }

  if (typeof document !== "undefined") {
    if (document.readyState === "loading") {
      document.addEventListener("DOMContentLoaded", upgradeGtkDropdowns);
    } else {
      upgradeGtkDropdowns();
    }
    document.addEventListener("click", closeAllPopovers);
    document.addEventListener("keydown", function (e) {
      if (e.key === "Escape") closeAllPopovers();
    });
    if (typeof window !== "undefined") {
      window.addEventListener("resize", closeAllPopovers);
      window.addEventListener("scroll", closeAllPopovers, true);
    }
  }

  global.brisa = brisa;
  connectSSE();
})(typeof window !== "undefined" ? window : globalThis);
`, a.token, a.config.AppID, a.config.Title, string(a.Theme()), initialOSTheme, string(commandsJSON))
}

// GenerateTypeScriptSDK generates full TypeScript declarations (.d.ts) for all registered Brisa commands.
func (a *App) GenerateTypeScriptSDK() string {
	spec := a.InspectorSpec()
	var sb strings.Builder
	sb.WriteString("// Auto-generated by Brisa Framework — DO NOT EDIT\n")
	sb.WriteString("// Application: " + spec.Title + " (" + spec.Version + ")\n\n")
	sb.WriteString("export interface BrisaCommandMap {\n")

	cmds := make([]CommandSchema, len(spec.Commands))
	copy(cmds, spec.Commands)
	sort.Slice(cmds, func(i, j int) bool { return cmds[i].Name < cmds[j].Name })

	for _, c := range cmds {
		inType := "void | Record<string, never>"
		if c.Input != nil && c.Input.TSType != "" {
			inType = c.Input.TSType
		}
		outType := "void"
		if c.Output != nil && c.Output.TSType != "" {
			outType = c.Output.TSType
		}
		if c.Summary != "" {
			fmt.Fprintf(&sb, "  /** %s */\n", c.Summary)
		}
		fmt.Fprintf(&sb, "  %q: { input: %s; output: %s };\n", c.Name, inType, outType)
	}
	sb.WriteString("}\n\n")
	sb.WriteString(`export interface BrisaSDK {
  readonly appId: string;
  readonly title: string;
  invoke<K extends keyof BrisaCommandMap>(
    command: K,
    payload?: BrisaCommandMap[K]["input"]
  ): Promise<BrisaCommandMap[K]["output"]>;
  on<T = any>(event: string, handler: (data: T) => void): () => void;
  once<T = any>(event: string, handler: (data: T) => void): () => void;
  emit(event: string, data?: any): Promise<void>;
}

declare global {
  interface Window {
    brisa: BrisaSDK;
  }
}
`)
	return sb.String()
}
