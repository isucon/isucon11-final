package scenario

type intData struct {
	tag string
	d   int64
}

type DebugData struct {
	ints  map[string][]int64
	intCh chan intData
}

func NewDebugData() *DebugData {
	d := &DebugData{
		ints:  map[string][]int64{},
		intCh: make(chan intData, 50),
	}

	go func() {
		for data := range d.intCh {
			d.ints[data.tag] = append(d.ints[data.tag], data.d)
		}
	}()

	return d
}

func (d *DebugData) AddInt(key string, data int64) {
	d.intCh <- intData{key, data}
}

func (d *DebugData) Close() {
	close(d.intCh)
}
