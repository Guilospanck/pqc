type GoToTUIMessageType =
  | "connected"
  | "disconnected"
  | "keys_exchanged"
  | "message"
  | "user_entered_chat"
  | "user_left_chat"
  | "current_users";
type TUIToGoMessageType = "connect" | "send";

export type TUIGoCommunication = {
  type: TUIToGoMessageType | GoToTUIMessageType;
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
