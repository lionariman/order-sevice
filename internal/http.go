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
	cache *Cache
	repo  *Repo
	cfg   *Config
}

func NewHTTP(cache *Cache, repo *Repo, cfg *Config) http.Handler {
	h := &HTTP{cache: cache, repo: repo, cfg: cfg}
	r := httprouter.New()
	r.GET("/order/:id", h.getOrder)
	r.ServeFiles("/static/*filepath", http.Dir("web"))
	r.GET("/", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		http.ServeFile(w, r, "web/index.html")
	})
	return r
}

func (h *HTTP) getOrder(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	start := time.Now()
	id := ps.ByName("id")

	// глобально выключенный кеш или ?nocache=1
	nocache := h.cfg == nil || !h.cfg.CacheEnabled || r.URL.Query().Has("nocache")

	if !nocache { // пробуем кеш
		if o, ok := h.cache.Get(id); ok {
			w.Header().Set("X-Source", "cache")
			w.Header().Set("X-Duration-ms", strconv.FormatInt(time.Since(start).Milliseconds(), 10))
			dur := time.Since(start)
			ms := float64(dur.Nanoseconds()) / 1e6
			w.Header().Set("X-Duration-ms", fmt.Sprintf("%.6f", ms))
			log.Printf("[HTTP] id=%s source=cache dur_ms=%.6f", id, ms)
			json.NewEncoder(w).Encode(o)
			return
		}
		log.Printf("[HTTP] cache-miss id=%s", id)
	}

	// идём в БД
	o, ok, err := h.repo.Get(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if !ok {
		http.NotFound(w, r)
		return
	}
	if !nocache {
		h.cache.Set(o)
	}

	w.Header().Set("X-Source", "db")
	w.Header().Set("X-Duration-ms", strconv.FormatInt(time.Since(start).Milliseconds(), 10))
	dur := time.Since(start)
	ms := float64(dur.Nanoseconds()) / 1e6
	log.Printf("[HTTP] id=%s source=db dur_ms=%.6f", id, ms)
	json.NewEncoder(w).Encode(o)
}
