package internal

import (
	"context"
	"encoding/json"
	"log"

	"github.com/IBM/sarama"
)

type Consumer struct {
	Group sarama.ConsumerGroup
	Topic string
	Cache OrderCache
	Repo  OrderRepository
}

func NewConsumer(brokers []string, topic, groupID string, cache OrderCache, repo OrderRepository) *Consumer {
	cfg := sarama.NewConfig()
	cfg.Version = sarama.V2_1_0_0
	cfg.Consumer.Return.Errors = true
	// cfg.Consumer.Offsets.Initial = sarama.OffsetNewest
	cfg.Consumer.Offsets.Initial = sarama.OffsetOldest

	cg, err := sarama.NewConsumerGroup(brokers, groupID, cfg)
	if err != nil {
		panic(err)
	}
	return &Consumer{
		Group: cg,
		Topic: topic,
		Cache: cache,
		Repo:  repo,
	}
}

func (c *Consumer) Start(ctx context.Context) error {
	handler := &CgHandler{Cache: c.Cache, Repo: c.Repo}
	for {
		if err := c.Group.Consume(ctx, []string{c.Topic}, handler); err != nil {
			log.Printf("Consume error: %v", err)
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}
}

func (c *Consumer) Close() error { return c.Group.Close() }

type CgHandler struct {
	Cache OrderCache
	Repo  OrderRepository
}

func (h *CgHandler) Setup(sarama.ConsumerGroupSession) error { return nil }

func (h *CgHandler) Cleanup(sarama.ConsumerGroupSession) error { return nil }

// Получаем партицию сообщений и обрабатываем их по одному в цикле
func (h *CgHandler) ConsumeClaim(sess sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		var o Order
		if err := json.Unmarshal(msg.Value, &o); err != nil {
			log.Printf("failed to decode order: %v", err)
			sess.MarkMessage(msg, "invalid-json")
			continue
		}
		if err := o.Validate(); err != nil {
			log.Printf("received invalid order: %v", err)
			sess.MarkMessage(msg, "invalid-data")
			continue
		}
		if err := h.Repo.Upsert(sess.Context(), &o); err != nil {
			continue
		}
		// каждый заказ кладем в бд
		// закомментировано потому что нам не следует класть каждый заказ в кеш
		// кладем в кеш только те последние которые клиент запросил
		// h.cache.Set(&o)
		// log.Printf("[CONSUMED] id=%s -> saved to DB and cache", o.OrderUID)
		sess.MarkMessage(msg, "")
	}
	return nil
}
