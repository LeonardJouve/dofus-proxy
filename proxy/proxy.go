package proxy

import (
	"dofus-proxy/config"
	"encoding/binary"
	"fmt"
	"net"
	"sync"

	"google.golang.org/protobuf/proto"
)

type Modifier *func(proto.Message) proto.Message
type Listener *func(proto.Message)

type proxy struct {
	sync.Mutex
	modifiers map[string]Modifier
	listeners map[string]Listener
	injector  chan proto.Message
}

const DOFUS_PORT = 5555

func New() *proxy {
	return &proxy{
		modifiers: make(map[string]Modifier),
		listeners: make(map[string]Listener),
		injector:  make(chan proto.Message),
	}
}

func (p *proxy) Listen(proxyPort uint16, configPort uint16) error {
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
			continue
		}

		fmt.Println("Client to server connected")

		serverToClient, err := net.Dial("tcp", originalHost)
		if err != nil {
			clientToServer.Close()
			continue
		}

		fmt.Println("Server to client connected")

		go p.handle(clientToServer, serverToClient)
		go p.handle(serverToClient, clientToServer)
	}
}

func (p *proxy) AddModifier(name string, modifier Modifier) {
	p.Lock()
	p.modifiers[name] = modifier
	p.Unlock()
}

func (p *proxy) RemoveModifier(name string) {
	p.Lock()
	delete(p.modifiers, name)
	p.Unlock()
}

func (p *proxy) AddListener(name string, listener Listener) {
	p.Lock()
	p.listeners[name] = listener
	p.Unlock()
}

func (p *proxy) RemoveListener(name string) {
	p.Lock()
	delete(p.listeners, name)
	p.Unlock()
}

func (p *proxy) Inject(message proto.Message) {
	p.injector <- message
}

func (p *proxy) handle(from net.Conn, to net.Conn) {
	defer from.Close()
	defer to.Close()
	var fragmentBuffer []byte

	for {
		buffer := make([]byte, 2048)

		n, err := from.Read(buffer)
		if err != nil {
			return
		}

		fragmentBuffer = append(fragmentBuffer, buffer[:n]...)

		for {
			size, sizeLength := binary.Uvarint(fragmentBuffer)
			if sizeLength <= 0 || uint64(len(fragmentBuffer)-sizeLength) < size {
				break
			}

			payload := fragmentBuffer[sizeLength : size+uint64(sizeLength)]

			fmt.Printf("Received %d bytes: %x\n", size, payload)

			// TODO: call listeners

			// TODO: handle modifiers

			fragmentBuffer = fragmentBuffer[uint64(sizeLength)+size:]
		}

		// TODO: send back modified message
		_, err = to.Write(buffer)
		if err != nil {
			return
		}

		done := false
		for !done {
			select {
			case message, ok := <-p.injector:
				if !ok {
					done = true
					break
				}

				// TODO: encode and send message
			default:
				done = true
			}
		}
	}
}
