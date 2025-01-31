package main

import (
	"dofus-proxy/config"
	connection "dofus-proxy/proto/connection/connection"
	game "dofus-proxy/proto/game/message"
	"dofus-proxy/proxy"
)

const (
	CONNECTION_PROXY_PORT = 5555
	GAME_PROXY_PORT       = 5556
	CONFIG_PORT           = 1234
)

func main() {
	originalHost, err := config.Run(CONNECTION_PROXY_PORT, CONFIG_PORT)
	if err != nil {
		return
	}

	connectionProxy := proxy.New(&connection.Message{})
	gameProxy := proxy.New(&game.Message{})
	connectionProxy.AddConnectionModifier(gameProxy, GAME_PROXY_PORT)
	connectionProxy.Listen(originalHost, CONNECTION_PROXY_PORT)
}
