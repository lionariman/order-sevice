package internal

import "sync"

// лимит на количество заказов в кеше
var MaxCacheLimit int = 10

type Cache struct {
	mu         sync.RWMutex
	m          map[string]*Order
	cacheLimit int
}

func NewCache() *Cache {
	return &Cache{
		m:          make(map[string]*Order),
		cacheLimit: MaxCacheLimit,
	}
}

func (c *Cache) Get(id string) (*Order, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	o, ok := c.m[id]
	return o, ok
}

func (c *Cache) Set(o *Order) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.m[o.OrderUID]; !ok && len(c.m) == c.cacheLimit {
		for k := range c.m {
			delete(c.m, k)
			break
		}
	}
	c.m[o.OrderUID] = o
}

func (c *Cache) Warm(list []*Order) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m = make(map[string]*Order)
	for i := 0; i < len(list) && i < c.cacheLimit; i++ {
		c.m[list[i].OrderUID] = list[i]
	}
}

func (c *Cache) Delete(orderUID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.m, orderUID)
}

func (c *Cache) DeleteAllItems() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m = make(map[string]*Order)
}
