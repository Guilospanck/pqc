#!/usr/bin/env bun

import {
  CliRenderer,
  createCliRenderer,
  TextRenderable,
  BoxRenderable,
  resolveRenderLib,
  KeyEvent,
  TextNodeRenderable,
} from "@opentui/core";

function setupCommonDemoKeys(renderer: CliRenderer) {
  renderer.keyInput.on("keypress", (key: KeyEvent) => {
    if (key.name === "`" || key.name === '"') {
      renderer.console.toggle();
    } else if (key.name === ".") {
      renderer.toggleDebugOverlay();
    } else if (key.name === "g" && key.ctrl) {
      console.log("dumping hit grid");
      renderer.dumpHitGrid();
    } else if (key.name === "l" && key.shift) {
      renderer.start();
    } else if (key.name === "s" && key.shift) {
      renderer.stop();
    } else if (key.name === "a" && key.shift) {
      renderer.auto();
    } else if (key.name === "a" && key.ctrl) {
      const lib = resolveRenderLib();
      const rawBytes = lib.getArenaAllocatedBytes();
      const formattedBytes = `${(rawBytes / 1024 / 1024).toFixed(2)} MB`;
      console.log("arena allocated bytes:", formattedBytes);
    }
  });
}

let mainContainer: BoxRenderable | null = null;
let messageArea: TextRenderable | null = null;
let inputBar: TextRenderable | null = null;
let statusText: TextRenderable | null = null;
let messages: Array<{ text: string; isSent: boolean; timestamp: Date }> = [];
let currentInput: string = "";
let inputCursorPosition: number = 0;

export function run(renderer: CliRenderer): void {
  renderer.setBackgroundColor("#0d1117");

  // Don't use a container, add directly to root for better control
  const rootBox = new BoxRenderable(renderer, {
    id: "rootBox",
    width: "100%",
    height: "100%",
    backgroundColor: "#161b22",
    zIndex: 1,
    border: false,
  });

  // Create message area that takes up most of the screen
  messageArea = new TextRenderable(renderer, {
    id: "messageArea",
    width: "100%",
    height: "85%", // Take 85% of height
    zIndex: 2,
    fg: "#f0f6fc",
  });
  rootBox.add(messageArea);

  // Create input bar at the bottom
  inputBar = new TextRenderable(renderer, {
    id: "inputBar",
    content: "> ",
    width: "100%",
    height: 5, // Fixed height of 5 lines
    zIndex: 3, // Higher z-index to appear on top
    fg: "#58a6ff",
  });
  rootBox.add(inputBar);

  // Create status area at the very bottom
  statusText = new TextRenderable(renderer, {
    id: "status",
    content: "Ready - Type a message and press Enter to send",
    width: "100%",
    height: 3, // Fixed height of 3 lines
    zIndex: 3, // Higher z-index to appear on top
    fg: "#8b949e",
  });
  rootBox.add(statusText);

  // Add some initial messages
  addMessage("Welcome to Chat TUI!", false);
  addMessage("Type your message and press Enter to send", false);
  addMessage("Your messages will appear on the right in blue", false);

  // Set up keyboard controls for chat
  renderer.keyInput.on("keypress", (event) => {
    const key = event.sequence;

    if (key === "\r" || key === "\n") {
      // Enter key - send message
      if (currentInput.trim()) {
        sendMessage();
      }
    } else if (key === "\u007f") {
      // Backspace
      if (inputCursorPosition > 0) {
        currentInput =
          currentInput.slice(0, inputCursorPosition - 1) +
          currentInput.slice(inputCursorPosition);
        inputCursorPosition--;
        updateInputBar();
      }
    } else if (key === "\u001b[D") {
      // Left arrow
      if (inputCursorPosition > 0) {
        inputCursorPosition--;
        updateInputBar();
      }
    } else if (key === "\u001b[C") {
      // Right arrow
      if (inputCursorPosition < currentInput.length) {
        inputCursorPosition++;
        updateInputBar();
      }
    } else if (key === "\u001b[H") {
      // Home key
      inputCursorPosition = 0;
      updateInputBar();
    } else if (key === "\u001b[F") {
      // End key
      inputCursorPosition = currentInput.length;
      updateInputBar();
    } else if (key === "\u001b" || event.name === "escape") {
      // Escape key - exit application
      destroy();
      renderer.stop();
      process.exit(0);
    } else if (
      key &&
      key.length === 1 &&
      !event.ctrl &&
      key !== "\r" &&
      key !== "\n" &&
      key !== "\u007f"
    ) {
      // Regular character input
      currentInput =
        currentInput.slice(0, inputCursorPosition) +
        key +
        currentInput.slice(inputCursorPosition);
      inputCursorPosition++;
      updateInputBar();
    }
  });

  updateInputBar();
  renderer.root.add(rootBox);
}

function sendMessage(): void {
  if (!currentInput.trim()) return;

  addMessage(currentInput, true);
  currentInput = "";
  inputCursorPosition = 0;
  updateInputBar();

  // Simulate a response after a short delay
  setTimeout(
    () => {
      const responses = [
        "That's interesting!",
        "Tell me more about that.",
        "I see what you mean.",
        "Thanks for sharing!",
        "How does that make you feel?",
        "That's a great point!",
        "I understand completely.",
      ];
      const randomResponse =
        responses[Math.floor(Math.random() * responses.length)];
      addMessage(randomResponse, false);
    },
    1000 + Math.random() * 2000,
  );
}

function addMessage(text: string, isSent: boolean): void {
  const message = {
    text: text,
    isSent: isSent,
    timestamp: new Date(),
  };

  messages.push(message);

  // Keep only last 50 messages
  if (messages.length > 50) {
    messages.shift();
  }

  updateMessageArea();
}

function updateMessageArea(): void {
  if (!messageArea) return;

  messageArea.clear();

  const messageNodes: TextNodeRenderable[] = [];

  // Add recent messages (show last 10 that fit in the area)
  const recentMessages = messages.slice(-10);

  recentMessages.forEach((msg) => {
    const timeStr = msg.timestamp.toLocaleTimeString([], {
      hour: "2-digit",
      minute: "2-digit",
    });

    if (msg.isSent) {
      // Sent message - right aligned, blue
      const messageNode = TextNodeRenderable.fromNodes([
        TextNodeRenderable.fromString(`${timeStr} `, { fg: "#8b949e" }),
        TextNodeRenderable.fromString("You: ", {
          fg: "#58a6ff",
          attributes: 1,
        }),
        TextNodeRenderable.fromString(msg.text, { fg: "#79c0ff" }),
      ]);
      messageNodes.push(messageNode);
    } else {
      // Received message - left aligned, green
      const messageNode = TextNodeRenderable.fromNodes([
        TextNodeRenderable.fromString(`${timeStr} `, { fg: "#8b949e" }),
        TextNodeRenderable.fromString("Them: ", {
          fg: "#56d364",
          attributes: 1,
        }),
        TextNodeRenderable.fromString(msg.text, { fg: "#7ee787" }),
      ]);
      messageNodes.push(messageNode);
    }

    // Add spacing between messages
    messageNodes.push(TextNodeRenderable.fromString("\n"));
  });

  const containerNode = TextNodeRenderable.fromNodes(messageNodes);
  messageArea.add(containerNode);
}

function updateInputBar(): void {
  if (!inputBar) return;

  // Create input display with cursor
  const beforeCursor = currentInput.slice(0, inputCursorPosition);
  const afterCursor = currentInput.slice(inputCursorPosition);
  const cursor = inputCursorPosition < currentInput.length ? "▊" : " ";

  // Use content property for simpler display
  inputBar.content = `> ${beforeCursor}${cursor}${afterCursor}`;

  // Update status
  if (statusText) {
    statusText.content =
      currentInput.length > 0
        ? `Type: ${currentInput.length} chars | Press Enter to send, ESC to exit`
        : "Ready - Type a message and press Enter to send, ESC to exit";
  }
}

export function destroy(): void {
  mainContainer?.destroyRecursively();
  mainContainer = null;
  messageArea = null;
  inputBar = null;
  statusText = null;
  messages = [];
  currentInput = "";
  inputCursorPosition = 0;
}

if (import.meta.main) {
  const renderer = await createCliRenderer({
    targetFps: 30,
    enableMouseMovement: true,
    exitOnCtrlC: true,
  });
  run(renderer);
  setupCommonDemoKeys(renderer);
}

