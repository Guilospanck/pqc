import { EventHandler } from "./singletons/event-handler";
import { State } from "./singletons/state";

export function isMessage(
  value?: unknown,
): value is { message: string; isSent: boolean } {
  return (
    value !== undefined &&
    value !== null &&
    typeof value === "object" &&
    "message" in value &&
    "isSent" in value &&
    typeof (value as any).message === "string" &&
    typeof (value as any).isSent === "boolean"
  );
}

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

  EventHandler().notify("update_message_area");
}
