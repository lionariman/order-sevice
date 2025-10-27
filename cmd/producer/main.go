package main

import (
	"encoding/json"
	"flag"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	intl "order-service/internal"

	"github.com/IBM/sarama"

	gofakeit "github.com/brianvoe/gofakeit/v7"
)

var rng *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))

func init() {
	gofakeit.Seed(time.Now().UnixNano())
}

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

	for i := 0; i < *n; i++ {
		o := genOrder()
		b, err := json.Marshal(o)
		if err != nil {
			log.Printf("marshall failed: %v", err)
			break
		}
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
	track := gofakeit.Regex("WB[A-Z0-9]{6}")
	items := genItems(track)

	return &intl.Order{
		OrderUID:    gofakeit.UUID(),
		TrackNumber: track,
		Entry:       "WBIL",
		Delivery: intl.Delivery{
			Name:    gofakeit.Name(),
			Phone:   gofakeit.Phone(),
			Zip:     gofakeit.Zip(),
			City:    gofakeit.City(),
			Address: gofakeit.Person().Address.Address,
			Region:  gofakeit.State(),
			Email:   gofakeit.Email(),
		},
		Payment: intl.Payment{
			Transaction:  gofakeit.UUID(), // как в примере
			RequestID:    "",
			Currency:     gofakeit.CurrencyShort(),
			Provider:     "wbpay",
			Amount:       int(gofakeit.Price(100, 10000)),
			PaymentDT:    gofakeit.DateRange(time.Unix(0, 0), time.Now()).Unix(),
			Bank:         gofakeit.BankName(),
			DeliveryCost: gofakeit.Number(100, 1000),
			GoodsTotal:   1,
			CustomFee:    1,
		},
		Items:             items,
		Locale:            "en",
		InternalSignature: "",
		CustomerID:        "test",
		DeliveryService:   "meest",
		ShardKey:          "9",
		SmID:              99,
		DateCreated:       gofakeit.Date(),
		OofShard:          "1",
	}
}

func genItems(track string) []intl.Item {
	n := 1 + rng.Intn(3) // 1..3 товаров
	items := make([]intl.Item, 0, n)
	for i := 0; i < n; i++ {
		items = append(items, intl.Item{
			ChrtID:      gofakeit.Number(100000, 999999),
			TrackNumber: track,
			Price:       int(gofakeit.Price(100, 10000)),
			RID:         gofakeit.UUID(),
			Name:        gofakeit.ProductName(),
			Sale:        gofakeit.RandomInt([]int{0, 10, 20, 30}),
			Size:        gofakeit.RandomString([]string{"0", "S", "M"}),
			TotalPrice:  1,
			NmID:        2000000 + gofakeit.RandomInt([]int{0, 900000}),
			Brand:       gofakeit.RandomString([]string{"Vivienne Sabo", "NoBrand", "Acme"}),
			Status:      gofakeit.RandomInt([]int{202, 208, 301}),
		})
	}
	return items
}
