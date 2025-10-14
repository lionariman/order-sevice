package internal

import (
	"context"
	"encoding/json"
	"log"

	"github.com/IBM/sarama"
)

type Consumer struct {
	group sarama.ConsumerGroup
	topic string
	cache *Cache
	repo  *Repo
}

func NewConsumer(brokers []string, topic, groupID string, cache *Cache, repo *Repo) *Consumer {
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
		group: cg,
		topic: topic,
		cache: cache,
		repo:  repo,
	}
}

func (c *Consumer) Start(ctx context.Context) error {
	handler := &cgHandler{cache: c.cache, repo: c.repo}
	for {
		if err := c.group.Consume(ctx, []string{c.topic}, handler); err != nil {
			log.Printf("Consume error: %v", err)
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}
}

func (c *Consumer) Close() error { return c.group.Close() }

type cgHandler struct {
	cache *Cache
	repo  *Repo
}

func (h *cgHandler) Setup(sarama.ConsumerGroupSession) error { return nil }

func (h *cgHandler) Cleanup(sarama.ConsumerGroupSession) error { return nil }

// Получаем партицию сообщений и обрабатываем их по одному в цикле
func (h *cgHandler) ConsumeClaim(sess sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		var o Order
		if err := json.Unmarshal(msg.Value, &o); err != nil || o.OrderUID == "" {
			sess.MarkMessage(msg, "invalid-json")
			continue
		}
		if err := h.repo.Upsert(sess.Context(), &o); err != nil {
			continue
		}
		h.cache.Set(&o)
		log.Printf("[CONSUMED] id=%s -> saved to DB and cache", o.OrderUID)
		sess.MarkMessage(msg, "")
	}
	return nil
}
