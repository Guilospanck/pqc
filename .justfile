build-go:
  cd core && go build .

start-tui: build-go
  cd tui && bun run dev
