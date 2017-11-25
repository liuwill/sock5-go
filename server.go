package sock5

import (
	"bufio"
	"errors"
	"log"
	"net"
	"time"
)

const (
	// Time allowed to read from or write a message to the peer.
	writeWait = 30 * time.Second
)

var processorController map[string]ProcessorBuilder

type ServerConfiguration struct {
	Mode     string // basic, auth, or ...
	OpenHttp bool
}

type Socks5Server struct {
	listener net.Listener
	clients  map[string]net.Conn

	builder ProcessorBuilder

	peers       map[string]*TcpConnection
	inPeerChan  chan *TcpConnection
	outPeerChan chan *TcpConnection
}

func NewSock5ServerConfigurable(address string, config ServerConfiguration) (*Socks5Server, error) {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, err
	}

	if _, ok := processorController[config.Mode]; !ok {
		return nil, errors.New("mode not exists")
	}

	log.Printf("server start at [ %s ]", address)
	socksServer := &Socks5Server{
		listener:    listener,
		builder:     processorController[config.Mode],
		peers:       make(map[string]*TcpConnection),
		clients:     make(map[string]net.Conn),
		inPeerChan:  make(chan *TcpConnection),
		outPeerChan: make(chan *TcpConnection),
	}
	go socksServer.Run()
	return socksServer, nil
}

func NewSocks5Server(address string) (*Socks5Server, error) {
	return NewSock5ServerConfigurable(address, ServerConfiguration{
		Mode:     "basic",
		OpenHttp: false,
	})
}

func (server *Socks5Server) HandleConnection(conn net.Conn) {
	addr := conn.RemoteAddr().String()

	tcpConnection := &TcpConnection{
		id:          GenerateAddrHash(addr),
		conn:        conn,
		reader:      bufio.NewReader(conn),
		server:      server,
		message:     make(chan []byte),
		processor:   server.builder(),
		targetProxy: make(map[string]*ProxyConnection),
	}

	go startTransform(tcpConnection)

	server.inPeerChan <- tcpConnection
}

func startTransform(tcpConnection *TcpConnection) {
	defer func() {
		tcpConnection.Close()
	}()

	go tcpConnection.Run()
	for {
		success := tcpConnection.processor.execute(tcpConnection)
		if !success {
			return
		}

		processor := tcpConnection.processor.nextProcessor()
		if processor == nil {
			return
		}
		tcpConnection.processor = processor

	}
}

func (server *Socks5Server) Run() {
	for {
		select {
		case inPeer := <-server.inPeerChan:
			server.peers[inPeer.id] = inPeer
			server.clients[inPeer.id] = inPeer.conn
		case outPeer := <-server.outPeerChan:
			delete(server.peers, outPeer.id)
			delete(server.clients, outPeer.id)
		}
	}
}

func (server *Socks5Server) Start() error {
	for {
		conn, err := server.listener.Accept()

		if err != nil {
			return err
		}

		// 异步处理，防止阻塞处理routine
		server.HandleConnection(conn)
	}
}

func installProcessorBuilder(key string, builder ProcessorBuilder) {
	if processorController == nil {
		processorController = make(map[string]ProcessorBuilder)
	}

	processorController[key] = builder
}
