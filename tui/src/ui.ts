import {
  BoxRenderable,
  TextNodeRenderable,
  TextRenderable,
} from "@opentui/core";
import { ClearState, State } from "./singletons/state";
import { COLORS } from "./constants";

let mainContainer: BoxRenderable | null = null;
let messageArea: TextRenderable | null = null;
let usersPanel: TextRenderable | null = null;
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
        TextNodeRenderable.fromString(`${timeStr} `, { fg: COLORS.timestamp }),
        TextNodeRenderable.fromString("You: ", {
          fg: msg.color,
          attributes: 1,
        }),
        TextNodeRenderable.fromString(msg.text, { fg: msg.color }),
      ]);
      messageNodes.push(messageNode);
    } else {
      // Received message - green
      const messageNode = TextNodeRenderable.fromNodes([
        TextNodeRenderable.fromString(`${timeStr} `, { fg: COLORS.timestamp }),
        TextNodeRenderable.fromString(msg.text, { fg: msg.color }),
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
    flexDirection: "row",
  });

  // Create main content area (70% width)
  const mainContentBox = new BoxRenderable(State.renderer, {
    id: "mainContentBox",
    width: "80%",
    height: "100%",
    backgroundColor: "#161b22",
    zIndex: 2,
    border: false,
  });

  // Create message area that takes up most of the screen height
  messageArea = new TextRenderable(State.renderer, {
    id: "messageArea",
    width: "100%",
    height: "85%",
    zIndex: 3,
    fg: "#f0f6fc",
  });
  mainContentBox.add(messageArea);

  // Create input bar at the bottom
  inputBar = new TextRenderable(State.renderer, {
    id: "inputBar",
    content: "> ",
    width: "100%",
    height: 5, // Fixed height of 5 lines
    zIndex: 4, // Higher z-index to appear on top
    fg: "#58a6ff",
  });
  mainContentBox.add(inputBar);

  // Create status area at the very bottom
  statusText = new TextRenderable(State.renderer, {
    id: "status",
    content: "Ready - Type a message and press Enter to send",
    width: "100%",
    height: 3, // Fixed height of 3 lines
    zIndex: 4, // Higher z-index to appear on top
    fg: "#8b949e",
  });
  mainContentBox.add(statusText);

  rootBox.add(mainContentBox);

  // Create users panel on the right (30% width)
  usersPanel = new TextRenderable(State.renderer, {
    id: "usersPanel",
    width: "20%",
    height: "85%",
    zIndex: 3,
    fg: "#f0f6fc",
    bg: "#0d1117",
  });
  rootBox.add(usersPanel);

  State.renderer.root.add(rootBox);
}

export function updateUsersPanel(): void {
  if (!usersPanel) return;

  usersPanel.clear();

  const userNodes: TextNodeRenderable[] = [];

  // Add header
  userNodes.push(
    TextNodeRenderable.fromString("Connected Users", {
      fg: "#58a6ff",
      attributes: 1,
    }),
  );
  userNodes.push(TextNodeRenderable.fromString("\n\n"));

  if (State.connectedUsers.length === 0) {
    userNodes.push(
      TextNodeRenderable.fromString("No users connected", {
        fg: "#8b949e",
      }),
    );
  } else {
    State.connectedUsers.forEach((user) => {
      const userNode = TextNodeRenderable.fromNodes([
        TextNodeRenderable.fromString("● ", {
          fg: user.color,
        }),
        TextNodeRenderable.fromString(user.username, {
          fg: user.color,
        }),
      ]);
      userNodes.push(userNode);
      userNodes.push(TextNodeRenderable.fromString("\n"));
    });
  }

  const containerNode = TextNodeRenderable.fromNodes(userNodes);
  usersPanel.add(containerNode);
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
  usersPanel = null;
  inputBar = null;
  statusText = null;
  ClearState();
}
