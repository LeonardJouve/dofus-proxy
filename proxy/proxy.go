package proxy

import (
	"bytes"
	connection "dofus-proxy/proto/connection/connection"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"reflect"
	"sync"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type Modifier func(proto.Message) (proto.Message, error)
type Listener func(proto.Message)
type Filter func(proto.Message) bool

type Proxy struct {
	sync.Mutex
	name           string
	messageType    proto.Message
	modifiers      map[string]Modifier
	listeners      map[string]Listener
	filters        map[string]Filter
	clientInjector chan proto.Message
	serverInjector chan proto.Message
}

const (
	CONNECTION_MODIFIER = "connection_modifier"
	LOG_LISTENER        = "log_listener"
)

func New(name string, messageType proto.Message) *Proxy {
	p := &Proxy{
		name:           name,
		messageType:    messageType,
		modifiers:      make(map[string]Modifier),
		listeners:      make(map[string]Listener),
		filters:        make(map[string]Filter),
		clientInjector: make(chan proto.Message),
		serverInjector: make(chan proto.Message),
	}

	return p
}

func (p *Proxy) Listen(host string, port uint16) error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}

	fmt.Printf("[%s] Proxy listening on %d\n", p.name, port)

	for {
		clientToServer, err := listener.Accept()
		fmt.Printf("[%s] New connection\n", p.name)
		if err != nil {
			continue
		}

		fmt.Printf("[%s] Client to server connected\n", p.name)

		serverToClient, err := net.Dial("tcp", host)
		if err != nil {
			clientToServer.Close()
			continue
		}

		fmt.Printf("[%s] Server to client connected\n", p.name)

		go p.handle(clientToServer, serverToClient, p.clientInjector)
		go p.handle(serverToClient, clientToServer, p.serverInjector)
	}
}

func (p *Proxy) AddModifier(name string, modifier Modifier) {
	p.Lock()
	p.modifiers[name] = modifier
	p.Unlock()
}

func (p *Proxy) RemoveModifier(name string) {
	p.Lock()
	delete(p.modifiers, name)
	p.Unlock()
}

func (p *Proxy) AddListener(name string, listener Listener) {
	p.Lock()
	p.listeners[name] = listener
	p.Unlock()
}

func (p *Proxy) RemoveListener(name string) {
	p.Lock()
	delete(p.listeners, name)
	p.Unlock()
}

func (p *Proxy) AddFilter(name string, filter Filter) {
	p.Lock()
	p.filters[name] = filter
	p.Unlock()
}

func (p *Proxy) RemoveFilter(name string) {
	p.Lock()
	delete(p.filters, name)
	p.Unlock()
}

func (p *Proxy) InjectClient(message proto.Message) {
	p.clientInjector <- message
}

func (p *Proxy) InjectServer(message proto.Message) {
	p.serverInjector <- message
}

func (p *Proxy) handle(from net.Conn, to net.Conn, injector chan proto.Message) {
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

			fmt.Printf("[%s] Received %d bytes: %x\n", p.name, size, payload)

			message, ok := reflect.New(reflect.TypeOf(p.messageType).Elem()).Interface().(proto.Message)
			if !ok {
				return
			}

			err := proto.Unmarshal(payload, message)
			if err != nil {
				return
			}

			p.Lock()
			for _, listener := range p.listeners {
				listener(message)
			}

			isFiltered := false
			for _, filter := range p.filters {
				if filter(message) {
					isFiltered = true
					break
				}
			}
			p.Unlock()

			if !isFiltered {
				verificationBuffer, err := proto.Marshal(message)
				if err != nil {
					return
				}

				if bytes.Equal(verificationBuffer, payload) {
					p.Lock()
					for _, modifier := range p.modifiers {
						newMessage, err := modifier(message)
						if err != nil {
							continue
						}
						message = newMessage
					}
					p.Unlock()

					data, err := encodeMessage(message)
					if err != nil {
						return
					}

					_, err = to.Write(data)
					if err != nil {
						return
					}
				} else {
					fmt.Printf("[%s] Message reencoded different from initial: skipping modifiers (probably outdated proto files)\n", p.name)

					_, err = to.Write(fragmentBuffer[:uint64(sizeLength)+size])
					if err != nil {
						return
					}
				}
			}

			fragmentBuffer = fragmentBuffer[uint64(sizeLength)+size:]
		}

		done := false
		for !done {
			select {
			case message, ok := <-injector:
				if !ok {
					return
				}

				data, err := encodeMessage(message)
				if err != nil {
					return
				}

				_, err = to.Write(data)
				if err != nil {
					return
				}
			default:
				done = true
			}
		}
	}
}

func encodeMessage(message proto.Message) ([]byte, error) {
	data, err := proto.Marshal(message)
	if err != nil {
		return []byte{}, err
	}

	varIntBuffer := make([]byte, binary.MaxVarintLen64)
	newSizeLength := binary.PutUvarint(varIntBuffer, uint64(len(data)))
	return append(varIntBuffer[:newSizeLength], data...), nil
}

func (p *Proxy) AddConnectionModifier(gameProxy *Proxy, port uint16) string {
	p.AddModifier(CONNECTION_MODIFIER, func(messsage proto.Message) (proto.Message, error) {
		connectionMessage, ok := messsage.(*connection.Message)
		if !ok {
			return nil, errors.New("not a connection message")
		}

		switch connectionMessage.Content.(type) {
		case *connection.Message_Response:
			response := connectionMessage.GetResponse()
			switch response.Content.(type) {
			case *connection.Response_SelectServer:
				selectServer := response.GetSelectServer()
				switch selectServer.Result.(type) {
				case *connection.SelectServerResponse_Success_:
					success := selectServer.GetSuccess()

					if len(success.GetPorts()) == 0 {
						return nil, errors.New("no port specified")
					}

					go gameProxy.Listen(fmt.Sprintf("%s:%d", success.GetHost(), success.GetPorts()[0]), port)

					success.Host = "localhost"
					success.Ports = []int32{int32(port)}
				}
			}
		}

		return connectionMessage, nil
	})

	return CONNECTION_MODIFIER
}

func (p *Proxy) AddLogListener() string {
	p.AddListener(LOG_LISTENER, func(message proto.Message) {
		json, err := protojson.MarshalOptions{Indent: "  "}.Marshal(message)
		if err != nil {
			return
		}

		fmt.Printf("[%s] Received:\n%s\n", p.name, string(json))
	})

	return LOG_LISTENER
}
