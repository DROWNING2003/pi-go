// pi-go TUI frontend using @earendil-works/pi-tui source
import { TUI, Text, Editor, Markdown, Loader, ProcessTerminal, matchesKey } from "../dist/index.js";
import { spawn } from "child_process";
import { fileURLToPath } from "url";
import path from "path";

const goBin = path.join(path.dirname(fileURLToPath(import.meta.url)), "..", "..", "bin", "pi");

// Spawn Go RPC backend
const backend = spawn(goBin, ["--rpc"], {
  stdio: ["pipe", "pipe", "inherit"],
  env: process.env,
});

// --- TUI setup ---
const terminal = new ProcessTerminal({});
const tui = new TUI(terminal);

// Welcome text
tui.addChild(new Text("pi ● Go backend + TypeScript TUI"));
tui.addChild(new Text(""));

// Input editor
const editor = new Editor(tui, {
  minHeight: 3,
  maxHeight: 10,
  placeholder: "Type a message... (/quit to exit)",
});

let isStreaming = false;
let buffer = "";

// Read responses from Go backend
backend.stdout.on("data", (chunk) => {
  buffer += chunk.toString();
  const lines = buffer.split("\n");
  buffer = lines.pop() || "";

  for (const line of lines) {
    if (!line.trim()) continue;
    try {
      const msg = JSON.parse(line);
      if (msg.type === "agent_end" && msg.messages) {
        for (const m of msg.messages) {
          if (m.role === "user") continue;
          if (m.role === "assistant" && m.content) {
            for (const b of m.content) {
              if (b.type === "text") {
                tui.addChild(new Text(""));
                tui.addChild(new Markdown(b.text, tui));
              } else if (b.type === "toolCall") {
                tui.addChild(new Text(`  [tool:${b.name}] ${JSON.stringify(b.arguments)}`));
              }
            }
          }
          if (m.role === "toolResult") {
            let text = "";
            for (const b of (m.content || [])) {
              if (b.type === "text") text += b.text;
            }
            if (text.length > 200) text = text.slice(0, 200) + "...";
            tui.addChild(new Text(`  [${m.toolName}] ${text}`));
          }
        }
        isStreaming = false;
        editor.value = "";
        editor.focus();
      }
    } catch {}
  }
});

// Handle editor submit
editor.onSubmit = (text) => {
  if (isStreaming) return;
  text = text.trim();
  if (!text) return;
  if (text === "/quit" || text === "/exit") {
    backend.stdin.write(JSON.stringify({ type: "quit" }) + "\n");
    tui.stop();
    process.exit(0);
  }

  isStreaming = true;
  tui.addChild(new Text(`▸ ${text}`));
  backend.stdin.write(JSON.stringify({ type: "prompt", message: text }) + "\n");
};

tui.addChild(editor);
tui.setFocus(editor);

// Ctrl+C to quit
tui.addInputListener((data) => {
  if (matchesKey(data, "ctrl+c")) {
    backend.stdin.write(JSON.stringify({ type: "quit" }) + "\n");
    tui.stop();
    process.exit(0);
  }
});

tui.start();
