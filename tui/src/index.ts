#!/usr/bin/env bun

import { createCliRenderer } from "@opentui/core";

import { execSync } from "node:child_process";
import { destroy, setupUI, updateInputBar, updateMessageArea } from "./ui";
import { sendToGo, setupGo } from "./go";
import { addMessage } from "./message";
import { State } from "./singleton";
import { setupKeyInputs } from "./keyListener";

function setup(): void {
  setupKeyInputs();
  setupUI();
  setupGo();

  // Add some initial messages
  addMessage("Welcome to Chat TUI!", false);
  addMessage("Type your message and press Enter to send", false);
  addMessage("Your messages will appear in blue", false);

  updateInputBar();
}

async function run(): Promise<void> {
  const renderer = await createCliRenderer({
    targetFps: 30,
    enableMouseMovement: true,
    exitOnCtrlC: true,
  });
  State.renderer = renderer;

  setup();
}

function sendMessage(): void {
  if (!State.currentInput.trim()) return;

  addMessage(State.currentInput, true);
  sendToGo("send", State.currentInput);

  State.currentInput = "";
  State.inputCursorPosition = 0;

  updateInputBar();
}

function exit(code?: number | null): void {
  if (!State.renderer) return;

  destroy();
  State.renderer.stop();

  try {
    execSync("clear", { stdio: "inherit" });
  } catch (e) {
    // Fallback if clear command fails
    process.stdout.write("\x1b[2J\x1b[H");
  }
  process.exit(code ?? 0);
}

function setupEventListeners(): void {
  globalThis.appEvents ??= new EventTarget();

  globalThis.appEvents.addEventListener("exit", (e: any) => {
    exit(e.detail.code);
  });

  globalThis.appEvents.addEventListener("update_message_area", () => {
    updateMessageArea();
  });

  globalThis.appEvents.addEventListener("send_message", () => {
    sendMessage();
  });

  globalThis.appEvents.addEventListener("update_input_bar", () => {
    updateInputBar();
  });

  globalThis.appEvents.addEventListener("add_message", (e: any) => {
    const { message, isSent } = e.detail as {
      message: string;
      isSent: boolean;
    };

    addMessage(message, isSent);
  });
}

if (import.meta.main) {
  setupEventListeners();
  run();
}
