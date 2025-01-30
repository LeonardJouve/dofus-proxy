package main

import (
	"dofus-proxy/config"
	connection "dofus-proxy/proto/connection/message"
	"dofus-proxy/proxy"
)

const (
	PROXY_PORT  = 10_000
	CONFIG_PORT = 1234
)

func main() {
	originalHost, err := config.Run(PROXY_PORT, CONFIG_PORT)
	if err != nil {
		return
	}

	p := proxy.New(originalHost, PROXY_PORT, &connection.Message{})
	p.Listen()
}
