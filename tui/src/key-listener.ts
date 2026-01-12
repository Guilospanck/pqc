import { EventHandler } from "./singletons/event-handler";
import { State } from "./singletons/state";

export function setupKeyInputs() {
  if (!State.renderer) return;

  const eventHandler = EventHandler();

  State.renderer.keyInput.on("keypress", (event) => {
    if (!State.renderer) return;
    const key = event.sequence;

    if (event.name === "`" || event.name === '"') {
      State.renderer.console.toggle();
    } else if (event.name === ".") {
      State.renderer.toggleDebugOverlay();
    } else if (key === "\r" || key === "\n") {
      // Enter key - send message
      if (State.currentInput.trim()) {
        eventHandler.notify("send_message");
      }
    } else if (key === "\u007f") {
      // Backspace
      if (State.inputCursorPosition > 0) {
        State.currentInput =
          State.currentInput.slice(0, State.inputCursorPosition - 1) +
          State.currentInput.slice(State.inputCursorPosition);
        State.inputCursorPosition--;
        eventHandler.notify("update_input_bar");
      }
    } else if (key === "\u001b[D") {
      // Left arrow
      if (State.inputCursorPosition > 0) {
        State.inputCursorPosition--;
        eventHandler.notify("update_input_bar");
      }
    } else if (key === "\u001b[C") {
      // Right arrow
      if (State.inputCursorPosition < State.currentInput.length) {
        State.inputCursorPosition++;
        eventHandler.notify("update_input_bar");
      }
    } else if (key === "\u001b[H") {
      // Home key
      State.inputCursorPosition = 0;
      eventHandler.notify("update_input_bar");
    } else if (key === "\u001b[F") {
      // End key
      State.inputCursorPosition = State.currentInput.length;
      eventHandler.notify("update_input_bar");
    } else if (key === "\u001b" || event.name === "escape") {
      // Escape key - exit application
      eventHandler.notify("exit");
    } else if (
      key &&
      key.length === 1 &&
      !event.ctrl &&
      key !== "\r" &&
      key !== "\n" &&
      key !== "\u007f"
    ) {
      // Regular character input
      State.currentInput =
        State.currentInput.slice(0, State.inputCursorPosition) +
        key +
        State.currentInput.slice(State.inputCursorPosition);
      State.inputCursorPosition++;
      eventHandler.notify("update_input_bar");
    }
  });
}
