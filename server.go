package sock5

import (
	"bufio"
	"fmt"
	"math/rand"
	"net"
	"time"
)

const (
	// Time allowed to read from or write a message to the peer.
	writeWait = 30 * time.Second
)

type Processor interface {
	execute(tcpConnection *TcpConnection) bool
	nextProcessor() Processor
}

type HandshakeProcess struct{}

func (processor *HandshakeProcess) execute(tcpConnection *TcpConnection) bool {
	buf := make([]byte, 3)
	n, err := tcpConnection.conn.Read(buf)
	// buf := new(bytes.Buffer)
	// payload, err := ioutil.ReadAll(tcpConnection.conn)
	// n, err := buf.Write(payload)

	if err != nil {
		return false
	}

	if buf[0] != 0x05 || buf[1] != 0x01 || buf[2] != 0x00 {
		println("disconnect", n)
		return false
	}

	tcpConnection.conn.Write([]byte{0x05, 0x00})
	// buffer := buf.Bytes()
	// if buf[0]
	// println(string(buf[0:1]), "=-=-=-")
	return true
}

func (processor *HandshakeProcess) nextProcessor() Processor {
	return nil
}

type TcpConnection struct {
	id        string
	conn      net.Conn
	reader    *bufio.Reader
	server    *Socks5Server
	message   chan []byte
	processor Processor
}

func (tcpConnection *TcpConnection) Close() {
	tcpConnection.server.outPeerChan <- tcpConnection
}

func (tcpConnection *TcpConnection) Read(buffer []byte) (int, error) {
	// tcpConnection.conn.SetReadDeadline(time.Now().Add(writeWait))
	targetBytes := make([]byte, 2048)
	n, err := tcpConnection.conn.Read(targetBytes)

	for i, v := range targetBytes[:n] {
		buffer[i] = v
	}
	return len(targetBytes), err
}

func (tcpConnection *TcpConnection) SendMessage(message []byte) {
	tcpConnection.message <- message
}

func (tcpConnection *TcpConnection) Run() {
	defer func() {
		tcpConnection.Close()
	}()

	for {
		message := <-tcpConnection.message
		tcpConnection.conn.SetWriteDeadline(time.Now().Add(writeWait))

		_, err := tcpConnection.conn.Write(message)
		if err != nil {
			return
		}
		// tcpConnection.conn.Write(targetBytes)
	}
}

type Socks5Server struct {
	listener net.Listener
	clients  map[string]net.Conn

	peers       map[string]*TcpConnection
	inPeerChan  chan *TcpConnection
	outPeerChan chan *TcpConnection
}

func NewSocks5Server(address string) (*Socks5Server, error) {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, err
	}

	fmt.Printf("server start at: %s", address)
	socksServer := &Socks5Server{
		listener:    listener,
		peers:       make(map[string]*TcpConnection),
		clients:     make(map[string]net.Conn),
		inPeerChan:  make(chan *TcpConnection),
		outPeerChan: make(chan *TcpConnection),
	}
	go socksServer.Run()
	return socksServer, nil
}

func (server *Socks5Server) HandleConnection(conn net.Conn) {
	tcpConnection := &TcpConnection{
		id:        randSeq(10),
		conn:      conn,
		reader:    bufio.NewReader(conn),
		server:    server,
		message:   make(chan []byte),
		processor: &HandshakeProcess{},
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
		// n, err := tcpConnection.Read(buffer)
		// if err != nil {
		// 	log.Println(fmt.Sprintf("Read message error: %s, session will be closed immediately", err.Error()))
		// 	return
		// }

		// if n <= 0 {
		// 	continue
		// }

		// fmt.Println(string(buffer[0:n]))
		// tcpConnection.SendMessage(buffer[0:n])
		// handler.handlerRequest(connSession, buffer, n)
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

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
