package main

import (
	"bytes"
	"dofus-proxy/config"
	"dofus-proxy/cosmetic"
	connection "dofus-proxy/proto/connection/connection"
	game "dofus-proxy/proto/game/message"
	"dofus-proxy/proxy"
	"os"
	"os/exec"
	"strings"
	"time"
)

const (
	CONNECTION_PROXY_PORT = 5555
	GAME_PROXY_PORT       = 5556
	CONFIG_PORT           = 1234
	DOFUS_PROCESS         = "Dofus.exe"
)

func isProcessRunning(process string) bool {
	cmd := exec.Command("tasklist")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return false
	}

	return strings.Contains(out.String(), process)
}

func main() {
	file, err := os.OpenFile("log.txt", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return
	}
	defer file.Close()

	os.Stdout = file

	undoRedirectZaap, err := config.RedirectZaap(CONFIG_PORT)
	if err != nil {
		return
	}
	defer undoRedirectZaap()

	originalHost, err := config.Run(CONNECTION_PROXY_PORT, CONFIG_PORT)
	if err != nil {
		return
	}

	connectionProxy := proxy.New("Connection", &connection.Message{})
	gameProxy := proxy.New("Game", &game.Message{})
	connectionProxy.AddLogListener()
	gameProxy.AddLogListener()
	cosmetic.AddCosmeticInventoryModifier(gameProxy)
	cosmetic.AddCosmeticInventoryEquipFilter(gameProxy)
	connectionProxy.AddConnectionModifier(gameProxy, GAME_PROXY_PORT)
	go connectionProxy.Listen(originalHost, CONNECTION_PROXY_PORT)

	for !isProcessRunning(DOFUS_PROCESS) {
		time.Sleep(5 * time.Second)
	}

	for isProcessRunning(DOFUS_PROCESS) {
		time.Sleep(5 * time.Second)
	}
}
