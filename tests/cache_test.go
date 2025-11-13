package tests

import (
	"fmt"
	"testing"

	intl "order-service/internal"

	"github.com/stretchr/testify/assert"
)

func TestCache_Eviction(t *testing.T) {
	cache := intl.NewCache()
	orders := make([]*intl.Order, 11)

	// создаю 11 заказов
	for i := range 11 {
		orders[i] = &intl.Order{OrderUID: fmt.Sprintf("%d", i+1)}
	}

	// добавляю 11 заказов в кеш
	for i := range 11 {
		cache.Set(orders[i])
	}

	// так как стоит ограничение на 10 заказов в кеше
	// проверяем что там сохранилось только 10 заказов, а не 11
	count := 0
	// считаю до 11 для проверки
	// хотя count должен быть равен 10
	for i := range 11 {
		idStr := fmt.Sprintf("%d", i+1)
		if _, ok := cache.Get(idStr); ok {
			assert.True(t, ok)
			count++
		}
	}
	assert.Equal(t, 10, count)
}

func TestCache_Warm(t *testing.T) {
	cache := intl.NewCache()
	orders := make([]*intl.Order, 15)

	// создаю 15 заказов
	for i := range 15 {
		orders[i] = &intl.Order{OrderUID: fmt.Sprintf("%d", i+1)}
	}

	// Warm сохраняет в кеш только 10 заказов
	cache.Warm(orders)

	count := 0
	for i := range 10 {
		idStr := fmt.Sprintf("%d", i+1)
		if _, ok := cache.Get(idStr); ok {
			assert.True(t, ok)
			count++
		}
	}
	assert.Equal(t, 10, count)
}

func TestCache_Delete(t *testing.T) {
	cache := intl.NewCache()
	orders := make([]*intl.Order, 15)

	// создаю 15 заказов
	for i := range 15 {
		orders[i] = &intl.Order{OrderUID: fmt.Sprintf("%d", i+1)}
	}

	// Warm сохраняет в кеш только 10 заказов
	// в этом тесте для удобства также использую Warm
	cache.Warm(orders)

	// удалим 3 заказа
	for i := range 3 {
		idStr := fmt.Sprintf("%d", i+1)
		cache.Delete(idStr)
	}
	assert.Equal(t, 7, len(cache.M))
}

func TestCache_DeleteAllItems(t *testing.T) {
	cache := intl.NewCache()
	orders := make([]*intl.Order, 15)

	// создаю 15 заказов
	for i := range 15 {
		orders[i] = &intl.Order{OrderUID: fmt.Sprintf("%d", i+1)}
	}

	// Warm сохраняет в кеш только 10 заказов
	// в этом тесте для удобства также использую Warm
	cache.Warm(orders)

	// удалим все заказы
	cache.DeleteAllItems()

	for i := range 1 {
		_, ok := cache.Get(fmt.Sprintf("%d", i+1))
		// проверим что заказов нет
		assert.False(t, ok)
	}
	assert.Equal(t, 0, len(cache.M))
}
