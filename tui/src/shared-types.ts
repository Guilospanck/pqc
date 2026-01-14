type GoToTUIMessageType =
  | "connected"
  | "keys_exchanged"
  | "message"
  | "user_entered_chat"
  | "user_left_chat";
type TUIToGoMessageType = "connect" | "send";

export type TUIGoCommunication = {
  type: TUIToGoMessageType | GoToTUIMessageType;
  value: string;
  color: string;
};

export type ConnectedUser = {
  username: string;
  color: string;
  joinedAt: Date;
};

export type TUIMessage = {
  text: string;
  isSent: boolean;
  timestamp: Date;
  color: string;
};
