import { spawn, type ChildProcessByStdio } from "node:child_process";
import type Stream from "node:stream";
import { type ConnectedUser, type TUIMessage } from "./types/shared-types";
import { EventHandler } from "./singletons/event-handler";
import {
  addConnectedUser,
  addMultipleConnectedUsers,
  addMultipleRooms,
  addRoom,
  removeConnectedUser,
  removeRoom,
  State,
  updateCurrentRoom,
} from "./singletons/state";
import type { RoomInfo, UIMessage } from "./types/generated-types";

let goProcess:
  | ChildProcessByStdio<Stream.Writable, Stream.Readable, Stream.Readable>
  | undefined = undefined;

const addMessage = (message: Omit<TUIMessage, "timestamp">) => {
  EventHandler().notify("add_message", { ...message });
};

export function setupGo(): void {
  // start go client
  goProcess = spawn("../core/client", [], {
    stdio: ["pipe", "pipe", "pipe"],
  });

  goProcess.on("exit", (code) => {
    EventHandler().notify("exit", { code });
  });

  goProcess.stdout.on("data", (data) => {
    const commands = String(data).trim().split("\n");

    for (const command of commands) {
      let message: UIMessage;
      try {
        message = JSON.parse(command);
      } catch {
        console.error("Error parsing data from Go stdout: ", command);
        return;
      }

      let tuiMessage: Omit<TUIMessage, "timestamp"> = {
        text: "",
        isSent: false,
        color: message.metadata.color,
      };

      switch (message.type) {
        case "connected": {
          State.currentUser = message.metadata;
          State.isConnected = true;

          addMessage({
            ...tuiMessage,
            text: "Connected to server.",
          });

          EventHandler().notify("update_current_user_text", {});
          EventHandler().notify("update_users_panel", {});
          break;
        }
        case "disconnected": {
          State.isConnected = false;
          State.connectedUsers = new Map();
          State.currentUser = message.metadata;

          addMessage({
            ...tuiMessage,
            text: "Disconnected from server.",
          });

          EventHandler().notify("update_current_user_text", {});
          EventHandler().notify("update_users_panel", {});
          break;
        }
        case "reconnecting": {
          addMessage({
            ...tuiMessage,
            text: "Reconnecting...",
          });

          break;
        }
        case "keys_exchanged": {
          addMessage({
            ...tuiMessage,
            text: "Keys exchanged.",
          });
          break;
        }
        case "message": {
          addMessage({
            ...tuiMessage,
            text: message.value,
          });
          break;
        }
        case "user_entered_chat": {
          addConnectedUser(message.metadata);
          EventHandler().notify("update_users_panel", {});
          break;
        }
        case "user_left_chat": {
          removeConnectedUser(message.metadata);
          EventHandler().notify("update_users_panel", {});
          break;
        }
        case "current_users": {
          let users: Array<ConnectedUser> = [];
          try {
            users = JSON.parse(message.value);
          } catch (err) {
            console.error(
              "Could not parse users from `current_users` event. Error: ",
              err,
            );
          }

          addMultipleConnectedUsers(users);
          EventHandler().notify("update_users_panel", {});
          break;
        }
        case "error":
        case "success": {
          addMessage({
            ...tuiMessage,
            text: message.value,
          });
          break;
        }
        case "joined_room": {
          let room: RoomInfo;
          try {
            room = JSON.parse(message.value);

            updateCurrentRoom(room);
            EventHandler().notify("update_rooms_panel", {});
          } catch (err) {
            console.error(
              "Could not parse room from `joined_room` event. Error: ",
              err,
            );
          }
          break;
        }
        case "created_room": {
          let room: RoomInfo;
          try {
            room = JSON.parse(message.value);

            addRoom(room);
            EventHandler().notify("update_rooms_panel", {});
          } catch (err) {
            console.error(
              "Could not parse room from `created_room` event. Error: ",
              err,
            );
          }
          break;
        }
        case "deleted_room": {
          let room: RoomInfo;
          try {
            room = JSON.parse(message.value);

            removeRoom(room);
            EventHandler().notify("update_rooms_panel", {});
          } catch (err) {
            console.error(
              "Could not parse room from `deleted_room` event. Error: ",
              err,
            );
          }
          break;
        }
        case "current_rooms": {
          let rooms: Array<RoomInfo> = [];
          try {
            rooms = JSON.parse(message.value);
          } catch (err) {
            console.error(
              "Could not parse rooms from `current_rooms` event. Error: ",
              err,
            );
          }

          addMultipleRooms(rooms);
          EventHandler().notify("update_rooms_panel", {});
          break;
        }
      }
    }
  });

  // Connects to WS server on startup
  sendToGo("connect", "");
}

// We talk to the go process via stdin/stdout
export function sendToGo(type: "connect" | "send", message: string) {
  if (!goProcess) return;

  const msg = {
    type,
    value: message,
  };

  goProcess.stdin.write(JSON.stringify(msg) + "\n");
}
