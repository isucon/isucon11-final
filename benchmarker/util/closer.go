package util

import (
	"sync"
	"time"
)

type Closer struct {
	closer     chan struct{}
	closeMutex sync.Mutex
	timerOnce  sync.Once
}

func NewCloser() *Closer {
	return &Closer{
		closer:     make(chan struct{}, 0),
		closeMutex: sync.Mutex{},
		timerOnce:  sync.Once{},
	}
}

func (c *Closer) WaitForClosing() <-chan struct{} {
	return c.closer
}

func (c *Closer) DoIfUnclosing(f func()) bool {
	c.closeMutex.Lock()
	defer c.closeMutex.Unlock()

	select {
	case _, _ = <-c.closer:
		f()
		return true
	default:
	}
	return false
}

func (c *Closer) CloseIfClosable() {
	c.closeMutex.Lock()
	defer c.closeMutex.Unlock()

	select {
	case _, _ = <-c.closer:
		// close済み
	default:
		close(c.closer)
	}
	return
}

func (c *Closer) CloseAfterTimeAtOnce(duration time.Duration) {
	c.timerOnce.Do(func() {
		time.Sleep(duration)
		c.CloseIfClosable()
	})
}
