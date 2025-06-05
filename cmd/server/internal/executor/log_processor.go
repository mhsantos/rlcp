package executor

type LogHandler interface {
	ProcessOutput([]byte) error
	RegisterListener(chan []byte)
	CloseListeners()
}
