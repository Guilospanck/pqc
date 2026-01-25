package main

import "time"

const PONG_WAIT = 10 * time.Second
const WRITE_WAIT = 5 * time.Second

// INFO: ping period needs to be less than pong wait, otherwise it will
// timeout the pong before we can ping
const PING_PERIOD = 5 * time.Second

var QUIT_COMMANDS = []string{
	"/quit",
	"/q",
	"/exit",
	":wq",
	":q",
	":wqa",
}

// How many reconnect attemps we are able to do
const MAX_ATTEMPTS int = 5
