import { spawn, type ChildProcessByStdio } from "node:child_process";
import type Stream from "node:stream";
import {
  type ConnectedUser,
  type TUIGoCommunication,
  type TUIMessage,
} from "./shared-types";
import { EventHandler } from "./singletons/event-handler";
import {
  addConnectedUser,
  addMultipleConnectedUsers,
  ClearState,
  removeConnectedUser,
  State,
} from "./singletons/state";

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
      let message: TUIGoCommunication;
      try {
        message = JSON.parse(command);
      } catch {
        console.error("Error parsing data from Go stdout: ", command);
        return;
      }

      let tuiMessage: Omit<TUIMessage, "timestamp"> = {
        text: "",
        isSent: false,
        color: message.color,
      };

      switch (message.type) {
        case "connected": {
          State.username = message.value;
          State.userColor = message.color;
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
          ClearState();
          State.username = message.value;
          State.userColor = message.color;

          addMessage({
            ...tuiMessage,
            text: "Disconnected from server.",
          });

          EventHandler().notify("update_current_user_text", {});
          EventHandler().notify("update_users_panel", {});
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
          addConnectedUser({ username: message.value, color: message.color });
          EventHandler().notify("update_users_panel", {});
          break;
        }
        case "user_left_chat": {
          removeConnectedUser({
            username: message.value,
            color: message.color,
          });
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
