package internal

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/julienschmidt/httprouter"
)

type HTTP struct {
	Cache OrderCache
	Repo  OrderRepository
	Cfg   *Config
}

func NewHTTP(cache OrderCache, repo OrderRepository, cfg *Config) http.Handler {
	h := &HTTP{Cache: cache, Repo: repo, Cfg: cfg}
	r := httprouter.New()
	r.GET("/order/:id", h.GetOrder)
	r.ServeFiles("/static/*filepath", http.Dir("web"))
	r.GET("/", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		http.ServeFile(w, r, "web/index.html")
	})
	return r
}

func (h *HTTP) GetOrder(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	start := time.Now()
	id := ps.ByName("id")

	// глобально выключенный кеш или ?nocache=1
	nocache := h.Cfg == nil || !h.Cfg.CacheEnabled || r.URL.Query().Has("nocache")

	if !nocache { // пробуем кеш
		if o, ok := h.Cache.Get(id); ok {
			w.Header().Set("X-Source", "cache")
			w.Header().Set("X-Duration-ms", strconv.FormatInt(time.Since(start).Milliseconds(), 10))
			dur := time.Since(start)
			ms := float64(dur.Nanoseconds()) / 1e6
			w.Header().Set("X-Duration-ms", fmt.Sprintf("%.6f", ms))
			log.Printf("[HTTP] id=%s source=cache dur_ms=%.6f", id, ms)

			// write data
			err := json.NewEncoder(w).Encode(o)
			if err != nil {
				log.Printf("Encoding error: %v", err)
			}
			return
		}
		log.Printf("[HTTP] cache-miss id=%s", id)
	}

	// идём в БД
	o, ok, err := h.Repo.Get(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		log.Printf("DB get request (no-cache) error: %v", err)
		return
	}
	if !ok {
		http.NotFound(w, r)
		log.Printf("DB not found (no-cache) error: %v", err)
		return
	}
	//
	if !nocache {
		h.Cache.Set(o)
	}

	w.Header().Set("X-Source", "db")
	w.Header().Set("X-Duration-ms", strconv.FormatInt(time.Since(start).Milliseconds(), 10))
	dur := time.Since(start)
	ms := float64(dur.Nanoseconds()) / 1e6
	log.Printf("[HTTP] id=%s source=db dur_ms=%.6f", id, ms)

	// write data
	// fmt.Println(">> DB [", o, "]")
	err = json.NewEncoder(w).Encode(o)
	if err != nil {
		log.Printf("Encoding (order to json) error: %v", err)
	}
}
