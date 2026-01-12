import { spawn, type ChildProcessByStdio } from "node:child_process";
import type Stream from "node:stream";
import { type UIMessage } from "./shared-types";
import { EventHandler } from "./singletons/event-handler";

let goProcess:
  | ChildProcessByStdio<Stream.Writable, Stream.Readable, Stream.Readable>
  | undefined = undefined;

const addMessage = (message: string, isSent: boolean) => {
  EventHandler().notify("add_message", { message, isSent });
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
    const message: UIMessage = JSON.parse(data);

    switch (message.type) {
      case "connected": {
        addMessage("Connected to server.", false);
        break;
      }
      case "keys_exchanged": {
        addMessage("Keys exchanged.", false);
        break;
      }
      case "message": {
        addMessage(message.value, false);
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
