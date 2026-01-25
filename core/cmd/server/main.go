package main

import (
	"context"
	"github.com/Guilospanck/pqc/core/pkg/logger"
)

func main() {
	logger.CreateMultiWriterLogger("ws-server-pqc")

	ctx := context.Background()

	server := NewServer(ctx)
	server.startServer()
}
