import type { MessageType } from "./generated-types";

export type TUIGoCommunication = {
  type: MessageType;
  value: string;
  color: string;
};

export type ConnectedUser = {
  username: string;
  color: string;
};

export type TUIMessage = {
  text: string;
  isSent: boolean;
  timestamp: Date;
  color: string;
};
