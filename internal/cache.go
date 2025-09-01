package internal

import "sync"

type Cache struct {
	mu sync.RWMutex
	m  map[string]*Order
}

func NewCache() *Cache { return &Cache{m: make(map[string]*Order)} }

func (c *Cache) Get(id string) (*Order, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	o, ok := c.m[id]
	return o, ok
}

func (c *Cache) Set(o *Order) {
	c.mu.Lock()
	c.m[o.OrderUID] = o
	c.mu.Unlock()
}

func (c *Cache) Warm(list []*Order) {
	c.mu.Lock()
	for _, o := range list {
		c.m[o.OrderUID] = o
	}
	c.mu.Unlock()
}
