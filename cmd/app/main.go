package main

import (
	"context"
	"log"
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
	go func() {
		err := consumer.Start(ctx)
		if err != nil {
			log.Printf("Consumer start error: %v", err)
		}
	}()
	defer consumer.Close()

	// http
	srv := &http.Server{
		Addr:         cfg.Addr,
		Handler:      intl.NewHTTP(cache, repo, &cfg),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	go func() {
		err := srv.ListenAndServe()
		if err != nil {
			log.Printf("Server start error: %v", err)
		}
	}()

	<-ctx.Done()
	shCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = srv.Shutdown(shCtx)
	if err != nil {
		log.Printf("shutdown error: %v", err)
	}
}
