package tests

import (
	"context"
	"encoding/json"
	"errors"
	"order-service/internal/mocks"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	intl "order-service/internal"
)

type mockSession struct {
	markedMessages []string
}

func (m *mockSession) MarkMessage(msg *sarama.ConsumerMessage, metadata string) {
	m.markedMessages = append(m.markedMessages, metadata)
}

func (m *mockSession) Context() context.Context {
	return context.Background()
}

func (m *mockSession) Claims() map[string][]int32 {
	return map[string][]int32{"orders": {0}} // Фейковые claims
}

func (m *mockSession) MemberID() string {
	return "test-member"
}

func (m *mockSession) GenerationID() int32 {
	return 1
}

func (m *mockSession) MarkOffset(topic string, partition int32, offset int64, metadata string) {}

func (m *mockSession) Commit() {}

func (m *mockSession) ResetOffset(topic string, partition int32, offset int64, metadata string) {}

func (m *mockSession) Close() error {
	return nil
}

type mockClaim struct {
	messages chan *sarama.ConsumerMessage
}

func (m *mockClaim) Messages() <-chan *sarama.ConsumerMessage {
	return m.messages
}

func (m *mockClaim) Topic() string {
	return "orders"
}

func (m *mockClaim) Partition() int32 {
	return 0
}

func (m *mockClaim) InitialOffset() int64 {
	return 0
}

func (m *mockClaim) HighWaterMarkOffset() int64 {
	return 100
}

func Test_ConsumeClaim_ValidMessage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCache := mocks.NewMockOrderCache(ctrl)
	mockRepo := mocks.NewMockOrderRepository(ctrl)

	handler := &intl.CgHandler{
		Cache: mockCache,
		Repo:  mockRepo,
	}

	sess := &mockSession{}
	claim := &mockClaim{
		messages: make(chan *sarama.ConsumerMessage, 1),
	}

	// Валидный JSON заказа
	validOrderJSON := `{"order_uid":"test-uid","track_number":"test-track","entry":"test-entry","delivery":{"name":"Test Name","phone":"+79999999999","zip":"123456","city":"Test City","address":"Test Address","region":"Test Region","email":"test@example.com"},"payment":{"transaction":"test-trans","request_id":"test-req","currency":"USD","provider":"test-prov","amount":1000,"payment_dt":1638360000,"bank":"test-bank","delivery_cost":100,"goods_total":900,"custom_fee":0},"items":[{"chrt_id":123,"track_number":"test-track","price":100,"rid":"test-rid","name":"Test Item","sale":0,"size":"M","total_price":100,"nm_id":456,"brand":"Test Brand","status":0}],"locale":"en","internal_signature":"test-sig","customer_id":"test-cust","delivery_service":"test-ds","shardkey":"test-shard","sm_id":789,"date_created":"2021-12-01T12:00:00Z","oof_shard":"test-oof"}`

	msg := &sarama.ConsumerMessage{
		Value: []byte(validOrderJSON),
	}

	claim.messages <- msg
	close(claim.messages)

	mockRepo.EXPECT().Upsert(gomock.Any(), gomock.Any()).Times(1)
	// mockCache.EXPECT().Upsert(gomock.Any(), gomock.Any()).Times(1)

	err := handler.ConsumeClaim(sess, claim)

	assert.NoError(t, err)
	assert.Contains(t, sess.markedMessages, "")
}

func Test_ConsumeClaim_NotValidMessage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCache := mocks.NewMockOrderCache(ctrl)
	mockRepo := mocks.NewMockOrderRepository(ctrl)

	handler := &intl.CgHandler{
		Cache: mockCache,
		Repo:  mockRepo,
	}

	sess := &mockSession{}
	claim := &mockClaim{
		messages: make(chan *sarama.ConsumerMessage, 1),
	}

	// Невалидный JSON заказа
	notValidOrderJSON := `{"order_uid":test-uid}`

	msg := &sarama.ConsumerMessage{
		Value: []byte(notValidOrderJSON),
	}

	claim.messages <- msg
	close(claim.messages)

	mockRepo.EXPECT().Upsert(gomock.Any(), gomock.Any()).Times(0)
	err := handler.ConsumeClaim(sess, claim)

	assert.NoError(t, err)
	assert.Contains(t, sess.markedMessages, "invalid-json")
}

func Test_ConsumeClaim_NotValidData(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCache := mocks.NewMockOrderCache(ctrl)
	mockRepo := mocks.NewMockOrderRepository(ctrl)

	handler := &intl.CgHandler{
		Cache: mockCache,
		Repo:  mockRepo,
	}

	sess := &mockSession{}
	claim := &mockClaim{
		messages: make(chan *sarama.ConsumerMessage, 1),
	}

	// Валидный JSON заказа с ошибкой
	notValidOrderData := `{"order_uid":"test-uid","track_number":"test-track","entry":"test-entry","delivery":{"name":"Test Name","phone":"+79999999999","zip":"123456","city":"Test City","address":"Test Address","email":"test@example.com"},"payment":{"transaction":"test-trans","request_id":"test-req","currency":"USD","provider":"test-prov","amount":1000,"payment_dt":1638360000,"bank":"test-bank","delivery_cost":100,"goods_total":900,"custom_fee":0},"items":[{"chrt_id":123,"track_number":"test-track","price":100,"rid":"test-rid","name":"Test Item","sale":0,"size":"M","total_price":100,"nm_id":456,"brand":"Test Brand","status":0}],"locale":"en","internal_signature":"test-sig","customer_id":"test-cust","delivery_service":"test-ds","shardkey":"test-shard","sm_id":789,"date_created":"2021-12-01T12:00:00Z","oof_shard":"test-oof"}`

	msg := &sarama.ConsumerMessage{
		Value: []byte(notValidOrderData),
	}

	claim.messages <- msg
	close(claim.messages)

	mockRepo.EXPECT().Upsert(gomock.Any(), gomock.Any()).Times(0)
	err := handler.ConsumeClaim(sess, claim)

	assert.NoError(t, err)
	assert.Contains(t, sess.markedMessages, "invalid-data")
}

func Test_ConsumeClaim_SuccessUpsert(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCache := mocks.NewMockOrderCache(ctrl)
	mockRepo := mocks.NewMockOrderRepository(ctrl)

	testOrder := intl.Order{
		OrderUID:    "test-uid-123",
		TrackNumber: "TN123456789",
		Entry:       "WBIL",
		Delivery: intl.Delivery{
			Name:    "John Doe",
			Phone:   "+71234567890",
			Zip:     "123456",
			City:    "Moscow",
			Address: "Lenina 1",
			Region:  "Moscow Oblast",
			Email:   "john@example.com",
		},
		Payment: intl.Payment{
			Transaction:  "txn-123",
			RequestID:    "req-456",
			Currency:     "RUB",
			Provider:     "wbpay",
			Amount:       1500,
			PaymentDT:    1638360000,
			Bank:         "Sberbank",
			DeliveryCost: 200,
			GoodsTotal:   1300,
			CustomFee:    0,
		},
		Items: []intl.Item{
			{
				ChrtID:      12345,
				TrackNumber: "TN123456789",
				Price:       500,
				RID:         "rid-789",
				Name:        "Test Item",
				Sale:        0,
				Size:        "L",
				TotalPrice:  500,
				NmID:        98765,
				Brand:       "Test Brand",
				Status:      0,
			},
		},
		Locale:            "ru",
		InternalSignature: "sig-abc",
		CustomerID:        "cust-123",
		DeliveryService:   "meest",
		ShardKey:          "shard-1",
		SmID:              123,
		DateCreated:       time.Date(2021, 12, 1, 0, 0, 0, 0, time.UTC),
		OofShard:          "oof-1",
	}

	mockRepo.EXPECT().Upsert(gomock.Any(), &testOrder).Return(nil).AnyTimes()
	// mockCache.EXPECT().Set(&testOrder).Return()

	handler := &intl.CgHandler{Cache: mockCache, Repo: mockRepo}
	sess := &mockSession{}
	claim := &mockClaim{messages: make(chan *sarama.ConsumerMessage, 3)}

	jsonTestOrder, _ := json.Marshal(testOrder)

	msg := &sarama.ConsumerMessage{Value: jsonTestOrder}
	msg2 := &sarama.ConsumerMessage{Value: jsonTestOrder}
	msg3 := &sarama.ConsumerMessage{Value: jsonTestOrder}

	claim.messages <- msg
	claim.messages <- msg2
	claim.messages <- msg3
	close(claim.messages)

	err := handler.ConsumeClaim(sess, claim)

	assert.NoError(t, err)
	assert.Contains(t, sess.markedMessages, "")
}

func Test_ConsumeClaim_ErrorUpsert(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockCache := mocks.NewMockOrderCache(ctrl)
	mockRepo := mocks.NewMockOrderRepository(ctrl)

	validOrderJSON := `{"order_uid":"test-uid","track_number":"test-track","entry":"test-entry","delivery":{"name":"Test Name","phone":"+79999999999","zip":"123456","city":"Test City","address":"Test Address","region":"Test Region","email":"test@example.com"},"payment":{"transaction":"test-trans","request_id":"test-req","currency":"USD","provider":"test-prov","amount":1000,"payment_dt":1638360000,"bank":"test-bank","delivery_cost":100,"goods_total":900,"custom_fee":0},"items":[{"chrt_id":123,"track_number":"test-track","price":100,"rid":"test-rid","name":"Test Item","sale":0,"size":"M","total_price":100,"nm_id":456,"brand":"Test Brand","status":0}],"locale":"en","internal_signature":"test-sig","customer_id":"test-cust","delivery_service":"test-ds","shardkey":"test-shard","sm_id":789,"date_created":"2021-12-01T12:00:00Z","oof_shard":"test-oof"}`

	mockRepo.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(errors.New("db error"))

	handler := &intl.CgHandler{Cache: mockCache, Repo: mockRepo}
	sess := &mockSession{}
	claim := &mockClaim{messages: make(chan *sarama.ConsumerMessage, 1)}

	msg := &sarama.ConsumerMessage{Value: []byte(validOrderJSON)}

	claim.messages <- msg
	close(claim.messages)

	err := handler.ConsumeClaim(sess, claim)

	assert.NoError(t, err)
	assert.NotContains(t, sess.markedMessages, "")
}
