import type { WSMetadata } from "./generated-types";

export type ConnectedUser = WSMetadata;

export type TUIMessage = {
  text: string;
  isSent: boolean;
  timestamp: Date;
  color: string;
};
