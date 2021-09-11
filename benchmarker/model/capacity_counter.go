package model

import (
	"sync"
)

// CapacityCounter はコマごとの、履修可能な学生数をトラックする
type CapacityCounter struct {
	capacity [5][6]int
	rmu      sync.RWMutex
}

func NewCapacityCounter() *CapacityCounter {
	return &CapacityCounter{}
}

func (c *CapacityCounter) Inc(dayOfWeek, period int) {
	c.rmu.Lock()
	defer c.rmu.Unlock()

	c.capacity[dayOfWeek][period]++
}

func (c *CapacityCounter) IncAll() {
	c.rmu.Lock()
	defer c.rmu.Unlock()

	for i := 0; i < 30; i++ {
		c.capacity[i/6][i%6]++
	}
}

func (c *CapacityCounter) Dec(dayOfWeek, period int) {
	c.rmu.Lock()
	defer c.rmu.Unlock()

	c.capacity[dayOfWeek][period]--
}

func (c *CapacityCounter) Get(dayOfWeek, period int) int {
	c.rmu.RLock()
	defer c.rmu.RUnlock()

	return c.capacity[dayOfWeek][period]
}
