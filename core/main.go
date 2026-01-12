package main

import (
	"pqc/client"
	"pqc/server"
)

func main() {
	go server.Run()
	client.Run()
}
