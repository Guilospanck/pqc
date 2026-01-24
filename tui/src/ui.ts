import {
  BoxRenderable,
  ScrollBoxRenderable,
  TextNodeRenderable,
  TextRenderable,
} from "@opentui/core";
import { ClearState, State } from "./singletons/state";
import { COLORS } from "./constants";

let messageArea: ScrollBoxRenderable | null = null;
let usersPanel: TextRenderable | null = null;
let inputBar: TextRenderable | null = null;
let statusText: TextRenderable | null = null;
let currentUserText: TextRenderable | null = null;

export function updateMessageArea(): void {
  if (!State.renderer || !messageArea) return;

  // Clear all existing children
  const children = messageArea.getChildren();
  children.forEach((child) => {
    messageArea!.remove(child.id);
  });

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

  // Create a TextRenderable to hold the content and add it to the scrollbox
  const textContent = new TextRenderable(State.renderer, {});
  textContent.add(containerNode);
  messageArea.add(textContent);
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

  // Create main content area
  const mainContentBox = new BoxRenderable(State.renderer, {
    id: "mainContentBox",
    width: "80%",
    height: "100%",
    backgroundColor: "#161b22",
    zIndex: 2,
    border: false,
  });

  // Create message area that takes up most of the screen height
  messageArea = new ScrollBoxRenderable(State.renderer, {
    id: "messageArea",
    stickyScroll: true,
    stickyStart: "bottom",
    scrollY: true,
    viewportCulling: true,
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

  const usersBox = new BoxRenderable(State.renderer, {
    id: "usersBox",
    width: "20%",
    height: "100%",
    backgroundColor: "#161b22",
    zIndex: 3,
    border: false,
  });

  // Create users panel on the right
  usersPanel = new TextRenderable(State.renderer, {
    id: "usersPanel",
    width: "100%",
    height: "95%",
    zIndex: 3,
    fg: "#f0f6fc",
    bg: "#0d1117",
  });
  usersBox.add(usersPanel);

  // show user status at bottom-right
  currentUserText = new TextRenderable(State.renderer, {
    id: "usersPanel",
    width: "100%",
    height: "5%",
    zIndex: 3,
    fg: "#f0f6fc",
    bg: "#0d1117",
  });
  usersBox.add(currentUserText);

  rootBox.add(usersBox);

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

  if (State.connectedUsers.size === 0) {
    userNodes.push(
      TextNodeRenderable.fromString("No users connected", {
        fg: "#8b949e",
      }),
    );
  } else {
    State.connectedUsers.forEach((user) => {
      if (user.username === State.username) return;

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

export function updateCurrentUser(): void {
  if (!currentUserText) return;

  currentUserText.clear();

  const userNode = TextNodeRenderable.fromNodes([
    TextNodeRenderable.fromString("● ", {
      fg: State.userColor,
    }),
    TextNodeRenderable.fromString(State.username, {
      fg: State.userColor,
    }),
  ]);

  const containerNode = TextNodeRenderable.fromNodes([userNode]);
  currentUserText.add(containerNode);
}

export function destroy(): void {
  messageArea = null;
  usersPanel = null;
  inputBar = null;
  statusText = null;
  ClearState();
}
