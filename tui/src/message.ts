import type { TUIMessage } from "./types/shared-types";
import { EventHandler } from "./singletons/event-handler";
import { State } from "./singletons/state";

export function isMessage(
  value?: unknown,
): value is Omit<TUIMessage, "timestamp"> {
  return (
    value !== undefined &&
    value !== null &&
    typeof value === "object" &&
    "text" in value &&
    "isSent" in value &&
    "color" in value &&
    typeof (value as any).text === "string" &&
    typeof (value as any).isSent === "boolean" &&
    typeof (value as any).color === "string"
  );
}

export function addMessage(msg: Omit<TUIMessage, "timestamp">): void {
  State.messages.push({
    ...msg,
    timestamp: new Date(),
  });

  // Keep only last 50 messages
  if (State.messages.length > 50) {
    State.messages.shift();
  }

  EventHandler().notify("update_message_area");
}
