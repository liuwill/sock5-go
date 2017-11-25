package sock5

type Processor interface {
	execute(tcpConnection *TcpConnection) bool
	nextProcessor() Processor
}
