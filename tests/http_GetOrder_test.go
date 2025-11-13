package tests

import (
	"errors"
	"net/http/httptest"
	"order-service/internal/mocks"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	intl "order-service/internal"
)

// TestHTTP_GetOrder_CacheHit проверяет сценарий, когда заказ найден в кеше.
// Ожидается:
// статус 200,
// заголовок X-Source: "cache".
func TestHTTP_GetOrder_CacheHit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCache := mocks.NewMockOrderCache(ctrl)
	mockRepo := mocks.NewMockOrderRepository(ctrl)
	testCfg := &intl.Config{CacheEnabled: true}
	h := &intl.HTTP{Cache: mockCache, Repo: mockRepo, Cfg: testCfg}
	order := &intl.Order{OrderUID: "test-ID"}

	mockCache.EXPECT().Get("test-ID").Return(order, true)

	req := httptest.NewRequest("GET", "/order/test-ID", nil)
	w := httptest.NewRecorder()
	h.GetOrder(w, req, []httprouter.Param{{Key: "id", Value: "test-ID"}})

	// t.Log(">>>>>", w.Header().Get("X-Source"))
	// t.Log(">>>>>", w.Header().Get("X-Duration-ms"))

	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "cache", w.Header().Get("X-Source"))
	assert.NotEmpty(t, w.Header().Get("X-Duration-ms"))

	t.Log("\n\n=====================\n")
}

// TestHTTP_GetOrder_Repo проверяет сценарий промаха в кеш,
// когда заказ берется из репозитория.
// Ожидается:
// статус 200,
// заголовок X-Source: "db",
// X-Duration-ms присутствует,
// заказ сохраняется в кеш.
func TestHTTP_GetOrder_Repo(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCache := mocks.NewMockOrderCache(ctrl)
	mockRepo := mocks.NewMockOrderRepository(ctrl)
	testCfg := &intl.Config{CacheEnabled: true}
	h := &intl.HTTP{Cache: mockCache, Repo: mockRepo, Cfg: testCfg}
	order := &intl.Order{OrderUID: "test-ID"}

	mockCache.EXPECT().Get("test-ID").Return(nil, false)
	mockRepo.EXPECT().Get(gomock.Any(), "test-ID").Return(order, true, nil)
	mockCache.EXPECT().Set(order)

	req := httptest.NewRequest("GET", "/order/test-ID", nil)
	w := httptest.NewRecorder()
	h.GetOrder(w, req, []httprouter.Param{{Key: "id", Value: "test-ID"}})

	t.Logf("Response code: %d", w.Code)
	t.Logf("X-Source header: %s", w.Header().Get("X-Source"))

	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "db", w.Header().Get("X-Source"))
	assert.NotEmpty(t, w.Header().Get("X-Duration-ms"))

	t.Log("\n\n=====================\n")
}

// TestHTTP_GetOrder_Repo_404_Error проверяет сценарий,
// когда заказ не найден в репозитории.
// Ожидается: статус 404.
func TestHTTP_GetOrder_Repo_404_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCache := mocks.NewMockOrderCache(ctrl)
	mockRepo := mocks.NewMockOrderRepository(ctrl)
	testCfg := &intl.Config{CacheEnabled: false}
	h := &intl.HTTP{Cache: mockCache, Repo: mockRepo, Cfg: testCfg}
	// order := &intl.Order{OrderUID: "test-ID"}

	// mockCache.EXPECT().Get("test-ID").Return(nil, false)
	mockRepo.EXPECT().Get(gomock.Any(), "test-ID").Return(nil, false, nil)
	// mockCache.EXPECT().Set(order)

	req := httptest.NewRequest("GET", "/order/test-ID", nil)
	w := httptest.NewRecorder()
	h.GetOrder(w, req, []httprouter.Param{{Key: "id", Value: "test-ID"}})

	assert.Equal(t, 404, w.Code)
	// assert.Equal(t, "db", w.Header().Get("X-Source"))
	// assert.NotEmpty(t, w.Header().Get("X-Duration-ms"))

	t.Log("\n\n=====================\n")
}

// TestHTTP_GetOrder_Repo_500_Error проверяет сценарий ошибки репозитория.
// Ожидается: статус 500.
func TestHTTP_GetOrder_Repo_500_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCache := mocks.NewMockOrderCache(ctrl)
	mockRepo := mocks.NewMockOrderRepository(ctrl)
	testCfg := &intl.Config{CacheEnabled: false}
	h := &intl.HTTP{Cache: mockCache, Repo: mockRepo, Cfg: testCfg}
	// order := &intl.Order{OrderUID: "test-ID"}

	// mockCache.EXPECT().Get("test-ID").Return(nil, false)
	mockRepo.EXPECT().Get(gomock.Any(), "test-ID").Return(nil, false, errors.New("db error"))
	// mockCache.EXPECT().Set(order)

	req := httptest.NewRequest("GET", "/order/test-ID", nil)
	w := httptest.NewRecorder()
	h.GetOrder(w, req, []httprouter.Param{{Key: "id", Value: "test-ID"}})

	assert.Equal(t, 500, w.Code)
	// assert.Equal(t, "db", w.Header().Get("X-Source"))
	// assert.NotEmpty(t, w.Header().Get("X-Duration-ms"))

	t.Log("\n\n=====================\n")
}

// TestHTTP_GetOrder_Repo_nocache1 проверяет сценарий с параметром ?nocache=1.
// Ожидается:
// статус 200,
// заголовок X-Source: "db", кеш игнорируется.
func TestHTTP_GetOrder_Repo_nocache1(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCache := mocks.NewMockOrderCache(ctrl)
	mockRepo := mocks.NewMockOrderRepository(ctrl)
	testCfg := &intl.Config{CacheEnabled: true}
	h := &intl.HTTP{Cache: mockCache, Repo: mockRepo, Cfg: testCfg}
	order := &intl.Order{OrderUID: "test-ID"}

	// mockCache.EXPECT().Get("test-ID").Return(nil, false)
	mockRepo.EXPECT().Get(gomock.Any(), "test-ID").Return(order, true, nil)
	// mockCache.EXPECT().Set(order)

	req := httptest.NewRequest("GET", "/order/test-ID?nocache=1", nil)
	w := httptest.NewRecorder()
	h.GetOrder(w, req, []httprouter.Param{{Key: "id", Value: "test-ID"}})

	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "db", w.Header().Get("X-Source"))
	// assert.NotEmpty(t, w.Header().Get("X-Duration-ms"))
}
