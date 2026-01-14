#!/usr/bin/env bun

import { createCliRenderer } from "@opentui/core";

import { execSync } from "node:child_process";
import { destroy, setupUI, updateInputBar, updateMessageArea, updateUsersPanel } from "./ui";
import { sendToGo, setupGo } from "./go";
import { addMessage, isMessage } from "./message";
import { State } from "./singletons/state";
import { setupKeyInputs } from "./key-listener";
import { EventHandler } from "./singletons/event-handler";
import { COLORS } from "./constants";

const EVENT_HANDLER_ID = "index.ts";

function setupEventListeners(): void {
  const eventHandler = EventHandler();

  eventHandler.subscribe("send_message", {
    id: EVENT_HANDLER_ID,
    callback() {
      sendMessage();
    },
  });

  eventHandler.subscribe("exit", {
    id: EVENT_HANDLER_ID,
    callback(code) {
      const codeNumber = Number(code);
      if (isNaN(codeNumber)) {
        exit();
      } else {
        exit(codeNumber);
      }
    },
  });

  eventHandler.subscribe("update_message_area", {
    id: EVENT_HANDLER_ID,
    callback() {
      updateMessageArea();
    },
  });

  eventHandler.subscribe("update_input_bar", {
    id: EVENT_HANDLER_ID,
    callback() {
      updateInputBar();
    },
  });

  eventHandler.subscribe("add_message", {
    id: EVENT_HANDLER_ID,
    callback(value) {
      if (!isMessage(value)) {
        console.error(
          'Expected value of type `Omit<TUIMessage, "timestamp"`. Received: ',
          value,
        );
        return;
      }

      addMessage(value);
    },
  });

  eventHandler.subscribe("update_users_panel", {
    id: EVENT_HANDLER_ID,
    callback() {
      updateUsersPanel();
    },
  });
}

function sendMessage(): void {
  if (!State.currentInput.trim()) return;

  addMessage({
    text: State.currentInput,
    isSent: true,
    color: COLORS.userMessage,
  });
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

async function run(): Promise<void> {
  const renderer = await createCliRenderer({
    targetFps: 30,
    enableMouseMovement: true,
    exitOnCtrlC: true,
  });
  State.renderer = renderer;

  setupKeyInputs();
  setupUI();
  setupGo();
  setupEventListeners();

  addMessage({
    text: "Welcome to Chat TUI!",
    isSent: false,
    color: COLORS.tuiMessage,
  });
  addMessage({
    text: "Type your message and press Enter to send",
    isSent: false,
    color: COLORS.tuiMessage,
  });
  addMessage({
    text: "Your messages will appear in blue",
    isSent: false,
    color: COLORS.tuiMessage,
  });

  updateInputBar();
}

if (import.meta.main) {
  run();
}
