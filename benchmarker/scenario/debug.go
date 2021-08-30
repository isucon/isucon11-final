package scenario

import "sync"

type DebugData struct {
	ints map[string][]int64

	isDebug bool
	mu      sync.Mutex
}

func NewDebugData(isDebug bool) *DebugData {
	return &DebugData{
		ints:    map[string][]int64{},
		isDebug: isDebug,
		mu:      sync.Mutex{},
	}
}

func (d *DebugData) AddInt(key string, data int64) {
	if !d.isDebug {
		return
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	d.ints[key] = append(d.ints[key], data)
}
