start-server: build-server
  ./core/server

start-client: build-client
  ./core/client

start-tui: build-client
  cd tui && bun run dev

build-client:
  cd core && go build ./cmd/client

build-server:
  cd core && go build ./cmd/server

