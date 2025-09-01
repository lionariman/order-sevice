package main

import (
	"encoding/hex"
	"encoding/json"
	"flag"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	intl "order-service/internal"

	"github.com/IBM/sarama"
)

func main() {
	// CLI flags
	n := flag.Int("n", 1, "how many orders to send")
	interval := flag.Duration("interval", time.Second, "interval between orders, e.g. 500ms, 1s, 2s")
	brokersFlag := flag.String("brokers", getenv("KAFKA_BROKERS", "localhost:29092"), "comma-separated kafka brokers")
	topic := flag.String("topic", getenv("KAFKA_TOPIC", "orders"), "kafka topic")
	flag.Parse()

	brokers := strings.Split(*brokersFlag, ",")
	cfg := sarama.NewConfig()
	cfg.Version = sarama.V2_1_0_0
	cfg.Producer.RequiredAcks = sarama.WaitForAll
	cfg.Producer.Return.Successes = true
	cfg.Producer.Retry.Max = 3

	prod, err := sarama.NewSyncProducer(brokers, cfg)
	if err != nil {
		log.Fatalf("producer create: %v", err)
	}
	defer prod.Close()

	rand.Seed(time.Now().UnixNano())

	for i := 0; i < *n; i++ {
		o := genOrder()
		b, _ := json.Marshal(o)
		msg := &sarama.ProducerMessage{
			Topic: *topic,
			Key:   sarama.StringEncoder(o.OrderUID),
			Value: sarama.ByteEncoder(b),
		}
		partition, offset, err := prod.SendMessage(msg)
		if err != nil {
			log.Printf("send failed: %v", err)
		} else {
			log.Printf("sent order_uid=%s partition=%d offset=%d", o.OrderUID, partition, offset)
		}
		if i+1 < *n {
			time.Sleep(*interval)
		}
	}
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func genOrder() *intl.Order {
	now := time.Now().UTC()
	orderUID := randHex(8) + randHex(8) // 32 hex chars

	track := "WB" + strings.ToUpper(randHex(6))
	items := genItems(track)

	goodsTotal := 0
	for _, it := range items {
		goodsTotal += it.TotalPrice
	}
	deliveryCost := 150 + rand.Intn(500)
	amount := goodsTotal + deliveryCost

	return &intl.Order{
		OrderUID:    orderUID,
		TrackNumber: track,
		Entry:       "WBIL",
		Delivery: intl.Delivery{
			Name:    pick([]string{"Test Testov", "Ivan Petrov", "John Doe"}),
			Phone:   "+7" + randomDigits(10),
			Zip:     pick([]string{"101000", "190000", "2639809"}),
			City:    pick([]string{"Moscow", "Kiryat Mozkin", "Berlin"}),
			Address: pick([]string{"Lenina 1", "Ploshad Mira 15", "Unter den Linden 5"}),
			Region:  pick([]string{"Moscow", "Kraiot", "Berlin"}),
			Email:   strings.ToLower(randHex(4)) + "@example.com",
		},
		Payment: intl.Payment{
			Transaction:  orderUID, // как в примере
			RequestID:    "",
			Currency:     pick([]string{"RUB", "USD", "EUR"}),
			Provider:     "wbpay",
			Amount:       amount,
			PaymentDT:    now.Unix(),
			Bank:         pick([]string{"alpha", "tbank", "sber"}),
			DeliveryCost: deliveryCost,
			GoodsTotal:   goodsTotal,
			CustomFee:    0,
		},
		Items:             items,
		Locale:            "en",
		InternalSignature: "",
		CustomerID:        "test",
		DeliveryService:   "meest",
		ShardKey:          "9",
		SmID:              99,
		DateCreated:       now,
		OofShard:          "1",
	}
}

func genItems(track string) []intl.Item {
	n := 1 + rand.Intn(3) // 1..3 товаров
	items := make([]intl.Item, 0, n)
	for i := 0; i < n; i++ {
		price := 100 + rand.Intn(900)
		sale := []int{0, 10, 20, 30}[rand.Intn(4)]
		total := int(float64(price) * (1 - float64(sale)/100))
		items = append(items, intl.Item{
			ChrtID:      900000 + rand.Intn(999999),
			TrackNumber: track,
			Price:       price,
			RID:         strings.ToLower(randHex(8) + randHex(4)),
			Name:        pick([]string{"Mascaras", "Socks", "T-Shirt"}),
			Sale:        sale,
			Size:        pick([]string{"0", "S", "M"}),
			TotalPrice:  total,
			NmID:        2000000 + rand.Intn(900000),
			Brand:       pick([]string{"Vivienne Sabo", "NoBrand", "Acme"}),
			Status:      pickInt([]int{202, 208, 301}),
		})
	}
	return items
}

func randHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		for i := range b {
			b[i] = byte(rand.Intn(256))
		}
	}
	return hex.EncodeToString(b)
}
func randomDigits(n int) string {
	sb := strings.Builder{}
	for i := 0; i < n; i++ {
		sb.WriteByte(byte('0' + rand.Intn(10)))
	}
	return sb.String()
}
func pick[T any](arr []T) T { return arr[rand.Intn(len(arr))] }
func pickInt(arr []int) int { return arr[rand.Intn(len(arr))] }
