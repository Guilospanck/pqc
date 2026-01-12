type GoToTUIMessageType = "connected" | "keys_exchanged" | "message";
type TUIToGoMessageType = "connect" | "send";

export type TUIGoCommunication = {
  type: TUIToGoMessageType | GoToTUIMessageType;
  value: string;
  color: string;
};

export type TUIMessage = {
  text: string;
  isSent: boolean;
  timestamp: Date;
  color: string;
};
