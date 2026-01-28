import type { CliRenderer } from "@opentui/core";
import type { TUIMessage, ConnectedUser } from "../types/shared-types";
import type { RoomInfo } from "../types/generated-types";

type ConnectedUserKey = string;
type RoomId = string;

type StateType = {
  renderer: CliRenderer | undefined;
  messages: Array<TUIMessage>;
  currentInput: string;
  inputCursorPosition: number;
  connectedUsers: Map<ConnectedUserKey, ConnectedUser>;
  availableRooms: Map<RoomId, RoomInfo>;
  currentRoom: RoomInfo | null;
  currentUser: ConnectedUser | null;
  isConnected: boolean;
};

export const State: StateType = {
  renderer: undefined,
  messages: [],
  currentInput: "",
  inputCursorPosition: 0,
  connectedUsers: new Map(),
  availableRooms: new Map(),
  currentRoom: null,
  currentUser: null,
  isConnected: false,
};

export function ClearState(): void {
  State.messages = [];
  State.currentInput = "";
  State.inputCursorPosition = 0;
  State.connectedUsers = new Map();
  State.availableRooms = new Map();
  State.currentUser = null;
  State.currentRoom = null;
  State.isConnected = false;
}

export function addMultipleConnectedUsers(users: Array<ConnectedUser>): void {
  for (const user of users) {
    State.connectedUsers.set(user.userId, user);
  }
}

export function addConnectedUser(user: ConnectedUser): void {
  if (user.userId === State.currentUser?.userId) return;
  State.connectedUsers.set(user.userId, user);
}

export function removeConnectedUser(user: ConnectedUser): void {
  if (user.userId === State.currentUser?.userId) return;

  State.connectedUsers.delete(user.userId);
}

export function addMultipleRooms(rooms: Array<RoomInfo>): void {
  State.availableRooms = new Map();
  for (const room of rooms) {
    State.availableRooms.set(room.ID, room);
  }
}

export function addRoom(room: RoomInfo): void {
  State.availableRooms.set(room.ID, room);
}

export function removeRoom(room: RoomInfo): void {
  State.availableRooms.delete(room.ID);
}

export function updateCurrentRoom(room: RoomInfo): void {
  State.currentRoom = room;
}
