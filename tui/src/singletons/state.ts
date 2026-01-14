import type { CliRenderer } from "@opentui/core";
import type { TUIMessage, ConnectedUser } from "../shared-types";

type StateType = {
  renderer: CliRenderer | undefined;
  messages: Array<TUIMessage>;
  currentInput: string;
  inputCursorPosition: number;
  connectedUsers: Array<ConnectedUser>;
};

export const State: StateType = {
  renderer: undefined,
  messages: [],
  currentInput: "",
  inputCursorPosition: 0,
  connectedUsers: [],
};

export function ClearState(): void {
  State.messages = [];
  State.currentInput = "";
  State.inputCursorPosition = 0;
  State.connectedUsers = [];
}

export function addConnectedUser(username: string, color: string): void {
  const existingUser = State.connectedUsers.find(user => user.username === username);
  if (!existingUser) {
    State.connectedUsers.push({
      username,
      color,
      joinedAt: new Date()
    });
  }
}

export function removeConnectedUser(username: string): void {
  State.connectedUsers = State.connectedUsers.filter(user => user.username !== username);
}
