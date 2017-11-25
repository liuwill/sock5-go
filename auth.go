package sock5

import (
	"log"
	"strconv"
)

type AuthHandshakeProcessor struct{}

func (processor *AuthHandshakeProcessor) execute(tcpConnection *TcpConnection) bool {
	buf := make([]byte, 1024)
	n, err := tcpConnection.conn.Read(buf)

	if err != nil || n < 3 {
		return false
	}

	if buf[0] != 0x05 {
		return false
	}
	methodLen := int(buf[1])

	methodBytes := buf[2 : methodLen+2]

	for _, method := range methodBytes {
		if method != 0x02 && method != 0x00 {
			return false
		}
	}

	tcpConnection.conn.Write([]byte{0x05, 0x02})
	return true
}

func (processor *AuthHandshakeProcessor) nextProcessor() Processor {
	return &AuthCheckProcessor{}
}

type AuthCheckProcessor struct{}

func (processor *AuthCheckProcessor) execute(tcpConnection *TcpConnection) bool {
	buf := make([]byte, 1024)
	n, err := tcpConnection.conn.Read(buf)

	if err != nil || n < 5 {
		return false
	}

	if buf[0] != 0x01 {
		return false
	}

	nameLen := int(buf[1])
	nameBytes := buf[2 : nameLen+2]
	username := string(nameBytes)

	passLen := int(buf[nameLen+2 : nameLen+3][0])
	passBytes := buf[nameLen+3 : passLen+nameLen+3]
	password := string(passBytes)

	if username != "will" || password != "111111" {
		return false
	}

	tcpConnection.conn.Write([]byte{0x01, 0x00})
	return true
}

func (processor *AuthCheckProcessor) nextProcessor() Processor {
	return &AuthRequestProcessor{}
}

type AuthRequestProcessor struct{}

func (processor *AuthRequestProcessor) execute(tcpConnection *TcpConnection) bool {
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

	log.Printf("socks connect establish from [%s] to [%s], Domain: [%s]", tcpConnection.conn.RemoteAddr().String(), proxyConnection.conn.RemoteAddr().String(), targetAddr)
	responseBytes := []byte{0x05, 0x00, 0x00, distType, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}

	tcpConnection.conn.Write(responseBytes)
	return true
}

func (processor *AuthRequestProcessor) nextProcessor() Processor {
	return nil
}

func init() {
	installProcessorBuilder("auth", func() Processor {
		return &AuthHandshakeProcessor{}
	})
}
