package main

import (
	"pqc/pkg/logger"
)

func main() {
	logger.CreateMultiWriterLogger("ws-server-pqc")
	server := WSServer{connections: nil}
	server.startServer()
}
