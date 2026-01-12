type ServerToUIMessageType = "connected" | "keys_exchanged" | "message";
type UIToServerMessageType = "connect" | "send";

export type UIMessage = {
  type: UIToServerMessageType | ServerToUIMessageType;
  value: string;
};
