import { spawn, type ChildProcessByStdio } from "node:child_process";
import type Stream from "node:stream";
import { type TUIGoCommunication, type TUIMessage } from "./shared-types";
import { EventHandler } from "./singletons/event-handler";

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
    let message: TUIGoCommunication;
    try {
      message = JSON.parse(data);
    } catch {
      console.error("Error parsing data from Go stdout: ", String(data));
      return;
    }

    let tuiMessage: Omit<TUIMessage, "timestamp"> = {
      text: "",
      isSent: false,
      color: message.color,
    };

    switch (message.type) {
      case "connected": {
        addMessage({
          ...tuiMessage,
          text: "Connected to server.",
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
