#!/usr/bin/env node
import { TUI, Text, Input, Markdown, ProcessTerminal, matchesKey } from "./src/index.ts";
import { spawn } from "node:child_process";
import { fileURLToPath } from "node:url";
import path from "node:path";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const goBin = path.join(__dirname, "..", "bin", "pi");

const backend = spawn(goBin, ["--rpc"], {
  stdio: ["pipe", "pipe", "inherit"],
  env: process.env,
});

const terminal = new ProcessTerminal({});
const tui = new TUI(terminal);

tui.addChild(new Text("pi  Go + TS TUI  /quit to exit"));
tui.addChild(new Text(""));

const input = new Input(tui, {
  placeholder: "Type a message...",
});

let streaming = false;
let buf = "";

backend.stdout.on("data", (chunk) => {
  buf += chunk.toString();
  const lines = buf.split("\n");
  buf = lines.pop() || "";
  for (const line of lines) {
    if (!line.trim()) continue;
    try {
      const msg = JSON.parse(line);
      if (msg.type === "agent_end" && msg.messages) {
        for (const m of msg.messages) {
          if (m.role === "assistant" && m.content) {
            for (const b of m.content) {
              if (b.type === "text") {
                tui.addChild(new Markdown(b.text, tui));
              } else if (b.type === "toolCall") {
                tui.addChild(new Text(`  [${b.name}] ${JSON.stringify(b.arguments)}`));
              }
            }
          }
        }
        streaming = false;
        input.value = "";
        input.focus();
      }
    } catch {}
  }
});

input.onSubmit = (text) => {
  if (streaming) return;
  text = text.trim();
  if (!text) return;
  if (text === "/quit" || text === "/exit") {
    backend.stdin.write(JSON.stringify({ type: "quit" }) + "\n");
    tui.stop();
    process.exit(0);
  }
  streaming = true;
  tui.addChild(new Text("▸ " + text));
  backend.stdin.write(JSON.stringify({ type: "prompt", message: text }) + "\n");
};

tui.addChild(input);
tui.setFocus(input);

tui.addInputListener((data) => {
  if (matchesKey(data, "ctrl+c")) {
    backend.stdin.write(JSON.stringify({ type: "quit" }) + "\n");
    tui.stop();
    process.exit(0);
  }
});

tui.start();
