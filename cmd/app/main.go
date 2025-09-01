package main

import (
	"context"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	intl "order-service/internal"
)

func main() {
	cfg := intl.Env()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.PGURL)
	if err != nil {
		panic(err)
	}
	defer pool.Close()

	repo := intl.NewRepo(pool)
	cache := intl.NewCache()

	// прогрев
	if list, err := repo.LoadRecent(ctx, cfg.WarmN); err == nil {
		cache.Warm(list)
	}

	// kafka consumer
	consumer := intl.NewConsumer(cfg.Brokers, cfg.Topic, cfg.Group, cache, repo)
	go func() { _ = consumer.Start(ctx) }()
	defer consumer.Close()

	// http
	srv := &http.Server{
		Addr:         cfg.Addr,
		Handler:      intl.NewHTTP(cache, repo, &cfg),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	go func() { _ = srv.ListenAndServe() }()

	<-ctx.Done()
	shCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(shCtx)
}
