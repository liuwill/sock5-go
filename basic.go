package sock5

import (
	"log"
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
	packBytes := make([]byte, 1024)
	n, err := tcpConnection.conn.Read(packBytes)

	if err != nil || n < 5 {
		return false
	}
	for i, val := range packBytes {
		if i >= n {
			break
		}
		print(int(val))
		print(" ")
	}
	println()
	headers := packBytes[0:4]

	if string(headers[:3]) != string([]byte{0x05, 0x01, 0x00}) {
		return false
	}
	distType := headers[3]

	var targetByte []byte
	portStart := 0
	switch distType {
	case 0x01:
		targetByte = packBytes[4:8] // make([]byte, 4)
		portStart = 8
	case 0x03:
		lengthBuf := packBytes[4:5] // make([]byte, 1)
		// _, err = tcpConnection.conn.Read(lengthBuf)
		// if err != nil {
		// 	return false
		// }

		addrLen := int(lengthBuf[0])
		targetByte = packBytes[5 : addrLen+5] // make([]byte, addrLen)
		portStart = addrLen + 5
	// case 0x04:
	default:
		return false
	}
	// n, err = tcpConnection.conn.Read(targetByte)
	// if err != nil {
	// 	return false
	// }

	targetAddr := string(targetByte)
	portBuffer := packBytes[portStart : portStart+2] // make([]byte, 2)
	// n, err = tcpConnection.conn.Read(portBuffer)
	targetPort := strconv.Itoa(int(portBuffer[0])<<8 | int(portBuffer[1]))

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

func (processor *RequestProcessor) nextProcessor() Processor {
	return nil
}

func init() {
	installProcessorBuilder("basic", func() Processor {
		return &HandshakeProcessor{}
	})
}
