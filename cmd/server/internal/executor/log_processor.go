package executor

type LogHandler interface {
	ProcessOutput([]byte)
	RegisterListener(chan []byte)
}
