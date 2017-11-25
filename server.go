package sock5

import (
	"bufio"
	"fmt"
	"io"
	"math/rand"
	"net"
	"strconv"
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

type HandshakeProcessor struct{}

func (processor *HandshakeProcessor) execute(tcpConnection *TcpConnection) bool {
	buf := make([]byte, 3)
	n, err := tcpConnection.conn.Read(buf)

	if err != nil {
		return false
	}

	if n != 3 || string(buf) != string([]byte{0x05, 0x01, 0x00}) {
		return false
	}

	tcpConnection.conn.Write([]byte{0x05, 0x00})
	return true
}

func (processor *HandshakeProcessor) nextProcessor() Processor {
	return &RequestProcessor{}
}

type RequestProcessor struct{}

func (processor *RequestProcessor) execute(tcpConnection *TcpConnection) bool {
	headers := make([]byte, 4)
	n, err := tcpConnection.conn.Read(headers)

	if err != nil {
		return false
	}

	if n != 4 || string(headers[:3]) != string([]byte{0x05, 0x01, 0x00}) {
		return false
	}
	distType := headers[3]

	var targetByte []byte
	switch distType {
	case 0x01:
		targetByte = make([]byte, 4)
	case 0x03:
		lengthBuf := make([]byte, 1)
		_, err = tcpConnection.conn.Read(lengthBuf)
		if err != nil {
			return false
		}

		addrLen := int(lengthBuf[0])
		targetByte = make([]byte, addrLen)
	// case 0x04:
	default:
		return false
	}
	n, err = tcpConnection.conn.Read(targetByte)
	if err != nil {
		return false
	}

	targetAddr := string(targetByte)
	portBuffer := make([]byte, 2)
	n, err = tcpConnection.conn.Read(portBuffer)
	targetPort := strconv.Itoa(int(portBuffer[0])<<8 | int(portBuffer[1]))

	// TODO create server proxy connection
	proxyConnection := NewProxyConnection(tcpConnection, targetAddr, targetPort)
	if proxyConnection == nil {
		return false
	}

	go tcpConnection.AddProxy(targetAddr, targetPort, proxyConnection)

	fmt.Printf("socks connect establish from [%s] to [%s], Domain: [%s] \n", tcpConnection.conn.RemoteAddr().String(), proxyConnection.conn.RemoteAddr().String(), targetAddr)
	responseBytes := []byte{0x05, 0x00, 0x00, distType, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}

	tcpConnection.conn.Write(responseBytes)
	return true
}

func (processor *RequestProcessor) nextProcessor() Processor {
	return nil
}

type ProxyConnection struct {
	address string
	port    string
	conn    net.Conn
	reader  *bufio.Reader
	client  *TcpConnection
	message chan []byte
}

func NewProxyConnection(tcpConnection *TcpConnection, targetAddr string, targetPort string) *ProxyConnection {
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%s", targetAddr, targetPort))
	if err != nil {
		return nil
	}

	return &ProxyConnection{
		conn:    conn,
		reader:  bufio.NewReader(conn),
		client:  tcpConnection,
		address: targetAddr,
		port:    targetPort,
		message: make(chan []byte),
	}
}

type TcpConnection struct {
	id          string
	conn        net.Conn
	reader      *bufio.Reader
	server      *Socks5Server
	message     chan []byte
	targetProxy map[string]*ProxyConnection
	processor   Processor
}

func (tcpConnection *TcpConnection) AddProxy(address string, port string, proxy *ProxyConnection) {
	key := fmt.Sprintf("%s:%s", address, port)
	tcpConnection.targetProxy[key] = proxy

	// TODO 处理Proxy，建立连接和全双工通信
	//进行转发
	go io.Copy(proxy.conn, tcpConnection.conn)
	io.Copy(tcpConnection.conn, proxy.conn)
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

	fmt.Printf("server start at [ %s ]", address)
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
		id:          randSeq(10),
		conn:        conn,
		reader:      bufio.NewReader(conn),
		server:      server,
		message:     make(chan []byte),
		processor:   &HandshakeProcessor{},
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
