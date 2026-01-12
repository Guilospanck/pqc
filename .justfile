build-go:
  cd core && go build .

start-tui: build-go
  cd tui && bun run dev

build-client:
  cd core && go build ./cmd/client

build-server:
  cd core && go build ./cmd/server

