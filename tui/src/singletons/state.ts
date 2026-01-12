import type { CliRenderer } from "@opentui/core";

type StateType = {
  renderer: CliRenderer | undefined;
  messages: Array<{
    text: string;
    isSent: boolean;
    timestamp: Date;
  }>;
  currentInput: string;
  inputCursorPosition: number;
};

export const State: StateType = {
  renderer: undefined,
  messages: [],
  currentInput: "",
  inputCursorPosition: 0,
};

export function ClearState(): void {
  State.messages = [];
  State.currentInput = "";
  State.inputCursorPosition = 0;
}
