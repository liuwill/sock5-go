package sock5

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"time"
)

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
