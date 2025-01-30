package proxy

import (
	"dofus-proxy/config"
	"encoding/binary"
	"fmt"
	"net"

	"google.golang.org/protobuf/proto"
)

type Modifier *func(proto.Message) proto.Message
type Listener *func(proto.Message)

type Proxy struct {
	modifiers []Modifier
	listeners []Listener
}

const DOFUS_PORT = 5555

func (proxy *Proxy) Listen(proxyPort uint16, configPort uint16) error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", proxyPort))
	if err != nil {
		return err
	}

	fmt.Printf("Proxy listening on %d\n", proxyPort)

	originalHost, err := config.Run(proxyPort, configPort)
	if err != nil {
		return err
	}

	for {
		clientToServer, err := listener.Accept()
		fmt.Println("New connection")
		if err != nil {
			return err
		}
		defer clientToServer.Close()

		fmt.Println("Client to server connected")

		serverToClient, err := net.Dial("tcp", originalHost)
		if err != nil {
			clientToServer.Close()
			return err
		}
		defer serverToClient.Close()

		fmt.Println("Server to client connected")

		go proxy.handle(clientToServer, serverToClient)
		go proxy.handle(serverToClient, clientToServer)
	}
}

func (proxy *Proxy) AddModifier(modifier Modifier) {
	proxy.modifiers = append(proxy.modifiers, modifier)
}

func (proxy *Proxy) AddListener(listener Listener) {
	proxy.listeners = append(proxy.listeners, listener)
}

func (proxy *Proxy) handle(from net.Conn, to net.Conn) {
	defer from.Close()
	defer to.Close()
	var fragmentBuffer []byte

	for {
		buffer := make([]byte, 2048)

		n, err := from.Read(buffer)
		if err != nil {
			return
		}

		_, err = to.Write(buffer)
		if err != nil {
			return
		}

		fragmentBuffer = append(fragmentBuffer, buffer[:n]...)

		size, sizeLength := binary.Uvarint(fragmentBuffer)
		if size == 0 || uint64(len(fragmentBuffer)-sizeLength) < size {
			continue
		}

		payload := fragmentBuffer[sizeLength : size+uint64(sizeLength)]

		fmt.Printf("Received %d bytes: %x\n", size, payload)

		fragmentBuffer = fragmentBuffer[uint64(sizeLength)+size:]
	}
}
