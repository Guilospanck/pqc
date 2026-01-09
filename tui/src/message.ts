import { State } from "./singleton";

type ServerToUIMessageType = "connected" | "keys_exchanged" | "message";
type UIToServerMessageType = "connect" | "send";

export type UIMessage = {
  type: UIToServerMessageType | ServerToUIMessageType;
  value: string;
};

export function addMessage(text: string, isSent: boolean): void {
  const message = {
    text: text,
    isSent: isSent,
    timestamp: new Date(),
  };

  State.messages.push(message);

  // Keep only last 50 messages
  if (State.messages.length > 50) {
    State.messages.shift();
  }

  globalThis.appEvents.dispatchEvent(
    new CustomEvent("update_message_area", {}),
  );
}
