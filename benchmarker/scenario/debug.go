package scenario

import "sync"

type DebugData struct {
	ints map[string][]int64

	mu sync.Mutex
}

func NewDebugData() *DebugData {
	return &DebugData{
		ints: map[string][]int64{},
		mu: sync.Mutex{},
	}
}

func (d *DebugData) AddInt(key string, data int64) {
	go func() {
		d.mu.Lock()
		defer d.mu.Unlock()

		d.ints[key] = append(d.ints[key], data)
	}()
}
