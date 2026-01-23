package main

import (
	"context"
	"pqc/pkg/logger"
)

func main() {
	logger.CreateMultiWriterLogger("ws-server-pqc")

	ctx := context.Background()

	server := NewServer(ctx)
	server.startServer()
}
