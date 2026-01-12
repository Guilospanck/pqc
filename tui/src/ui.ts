import {
  BoxRenderable,
  TextNodeRenderable,
  TextRenderable,
} from "@opentui/core";
import { ClearState, State } from "./singletons/state";

let mainContainer: BoxRenderable | null = null;
let messageArea: TextRenderable | null = null;
let inputBar: TextRenderable | null = null;
let statusText: TextRenderable | null = null;

export function updateMessageArea(): void {
  if (!messageArea) return;

  messageArea.clear();

  const messageNodes: TextNodeRenderable[] = [];

  const recentMessages = State.messages.slice(-100);

  recentMessages.forEach((msg) => {
    const timeStr = msg.timestamp.toLocaleTimeString([], {
      hour: "2-digit",
      minute: "2-digit",
    });

    if (msg.isSent) {
      // Sent message - blue
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
      // Received message - green
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

export function setupUI(): void {
  if (!State.renderer) return;

  State.renderer.setBackgroundColor("#0d1117");

  const rootBox = new BoxRenderable(State.renderer, {
    id: "rootBox",
    width: "100%",
    height: "100%",
    backgroundColor: "#161b22",
    zIndex: 1,
    border: false,
  });

  // Create message area that takes up most of the screen
  messageArea = new TextRenderable(State.renderer, {
    id: "messageArea",
    width: "100%",
    height: "85%",
    zIndex: 2,
    fg: "#f0f6fc",
  });
  rootBox.add(messageArea);

  // Create input bar at the bottom
  inputBar = new TextRenderable(State.renderer, {
    id: "inputBar",
    content: "> ",
    width: "100%",
    height: 5, // Fixed height of 5 lines
    zIndex: 3, // Higher z-index to appear on top
    fg: "#58a6ff",
  });
  rootBox.add(inputBar);

  // Create status area at the very bottom
  statusText = new TextRenderable(State.renderer, {
    id: "status",
    content: "Ready - Type a message and press Enter to send",
    width: "100%",
    height: 3, // Fixed height of 3 lines
    zIndex: 3, // Higher z-index to appear on top
    fg: "#8b949e",
  });
  rootBox.add(statusText);

  State.renderer.root.add(rootBox);
}

export function updateInputBar(): void {
  if (!inputBar) return;

  // Create input display with cursor
  const beforeCursor = State.currentInput.slice(0, State.inputCursorPosition);
  const afterCursor = State.currentInput.slice(State.inputCursorPosition);
  const cursor =
    State.inputCursorPosition < State.currentInput.length ? "▊" : " ";

  // Use content property for simpler display
  inputBar.content = `> ${beforeCursor}${cursor}${afterCursor}`;

  // Update status
  if (statusText) {
    statusText.content =
      State.currentInput.length > 0
        ? `Type: ${State.currentInput.length} chars | Press Enter to send, ESC to exit`
        : "Ready - Type a message and press Enter to send, ESC to exit";
  }
}

export function destroy(): void {
  mainContainer?.destroyRecursively();
  mainContainer = null;
  messageArea = null;
  inputBar = null;
  statusText = null;
  ClearState();
}
