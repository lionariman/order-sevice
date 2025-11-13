package internal

import "sync"

// лимит на количество заказов в кеше
var MaxCacheLimit int = 10

type Cache struct {
	Mu         sync.RWMutex
	M          map[string]*Order
	CacheLimit int
}

func NewCache() *Cache {
	return &Cache{
		M:          make(map[string]*Order),
		CacheLimit: MaxCacheLimit,
	}
}

func (c *Cache) Get(id string) (*Order, bool) {
	c.Mu.RLock()
	defer c.Mu.RUnlock()
	o, ok := c.M[id]
	return o, ok
}

func (c *Cache) Set(o *Order) {
	c.Mu.Lock()
	defer c.Mu.Unlock()
	if _, ok := c.M[o.OrderUID]; !ok && len(c.M) == c.CacheLimit {
		for k := range c.M {
			delete(c.M, k)
			break
		}
	}
	c.M[o.OrderUID] = o
}

func (c *Cache) Warm(list []*Order) {
	c.Mu.Lock()
	defer c.Mu.Unlock()
	c.M = make(map[string]*Order)
	for i := 0; i < len(list) && i < c.CacheLimit; i++ {
		c.M[list[i].OrderUID] = list[i]
	}
}

func (c *Cache) Delete(orderUID string) {
	c.Mu.Lock()
	defer c.Mu.Unlock()
	delete(c.M, orderUID)
}

func (c *Cache) DeleteAllItems() {
	c.Mu.Lock()
	defer c.Mu.Unlock()
	c.M = make(map[string]*Order)
}
