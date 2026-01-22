import type { CliRenderer } from "@opentui/core";
import type { TUIMessage, ConnectedUser } from "../shared-types";

type ConnectedUserKey = string;

type StateType = {
  renderer: CliRenderer | undefined;
  messages: Array<TUIMessage>;
  currentInput: string;
  inputCursorPosition: number;
  connectedUsers: Map<ConnectedUserKey, ConnectedUser>;
  username: string;
  userColor: string;
  isConnected: boolean;
};

export const State: StateType = {
  renderer: undefined,
  messages: [],
  currentInput: "",
  inputCursorPosition: 0,
  connectedUsers: new Map(),
  username: "",
  userColor: "",
  isConnected: false,
};

export function ClearState(): void {
  State.messages = [];
  State.currentInput = "";
  State.inputCursorPosition = 0;
  State.connectedUsers = new Map();
  State.username = "";
  State.userColor = "";
  State.isConnected = false;
}

function key(connectedUser: ConnectedUser): string {
  return `${connectedUser.username}:${connectedUser.color}`;
}

export function addMultipleConnectedUsers(users: Array<ConnectedUser>): void {
  for (const user of users) {
    State.connectedUsers.set(key(user), user);
  }
}

export function addConnectedUser(user: ConnectedUser): void {
  if (user.username === State.username) return;
  State.connectedUsers.set(key(user), user);
}

// FIXME: this can be a problem if the server generates the same username
// for more than one user.
export function removeConnectedUser(user: ConnectedUser): void {
  if (user.username === State.username) return;

  State.connectedUsers.delete(key(user));
}
