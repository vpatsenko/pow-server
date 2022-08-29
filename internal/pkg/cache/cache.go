package cache

import (
	"fmt"
	"time"

	"github.com/patrickmn/go-cache"
)

type Cache struct {
	cache *cache.Cache
}

func NewCache() *Cache {
	return &Cache{cache: cache.New(5*time.Minute, 10*time.Minute)}
}
func (c *Cache) Add(key int, expiration int64) error {
	return c.cache.Add(fmt.Sprint(key), "val", cache.DefaultExpiration)
}

func (c *Cache) Get(key int) bool {
	_, ok := c.cache.Get(fmt.Sprint(key))
	return ok
}
func (c *Cache) Delete(key int) {
	c.cache.Delete(fmt.Sprint(key))
}
