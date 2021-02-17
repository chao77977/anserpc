package anserpc

type waitProc interface {
	wait() error
	stop()
}
