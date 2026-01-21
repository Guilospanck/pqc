import type { CliRenderer } from "@opentui/core";
import type { TUIMessage, ConnectedUser } from "../shared-types";

type StateType = {
  renderer: CliRenderer | undefined;
  messages: Array<TUIMessage>;
  currentInput: string;
  inputCursorPosition: number;
  connectedUsers: Array<ConnectedUser>;
  username: string;
  userColor: string;
  isConnected: boolean;
};

export const State: StateType = {
  renderer: undefined,
  messages: [],
  currentInput: "",
  inputCursorPosition: 0,
  connectedUsers: [],
  username: "",
  userColor: "",
  isConnected: false,
};

export function ClearState(): void {
  State.messages = [];
  State.currentInput = "";
  State.inputCursorPosition = 0;
  State.connectedUsers = [];
  State.username = "";
  State.userColor = "";
  State.isConnected = false;
}

export function addMultipleConnectedUsers(users: Array<ConnectedUser>): void {
  State.connectedUsers.push(...users);
}

export function addConnectedUser(username: string, color: string): void {
  if (username === State.username) return;

  const existingUser = State.connectedUsers.find(
    (user) => user.username === username,
  );

  if (existingUser) return;

  State.connectedUsers.push({
    username,
    color,
  });
}

// FIXME: this can be a problem if the server generates the same username
// for more than one user.
export function removeConnectedUser(username: string): void {
  if (username === State.username) return;

  State.connectedUsers = State.connectedUsers.filter(
    (user) => user.username !== username,
  );
}
