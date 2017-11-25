package sock5

import (
	"fmt"
	"strconv"
)

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
