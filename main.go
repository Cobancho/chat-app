package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"sistema-maika-chat/clients"
	"sistema-maika-chat/handlers"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func root(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("The Maika Chat is listening!"))
}

func main() {
	dbClient, err := sqlx.Connect("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal("Error conecting to postgres", err)
	}
	clients.DB = dbClient

	defer clients.DB.Close()

	if err := clients.DB.Ping(); err != nil {
		log.Fatal("Could not ping to postgres", err)
	} else {
		log.Println("Succesfully connected to Postgres")
	}

	opt, err := redis.ParseURL(os.Getenv("REDIS_URL"))
	if err != nil {
		log.Fatal("Error parsing redis url", err)
	}

	clients.Redis = redis.NewClient(opt)

	if err := redisotel.InstrumentTracing(clients.Redis); err != nil {
		log.Panic(err)
	}

	if err := redisotel.InstrumentMetrics(clients.Redis); err != nil {
		log.Panic(err)
	}

	ctx := context.Background()

	response, err := clients.Redis.Ping(ctx).Result()

	if err != nil {
		log.Fatal("Could not ping to redis", err)
	} else {
		log.Println("Succesfully connected to redis", response)
	}

	router := http.NewServeMux()

	router.HandleFunc("/", root)
	router.HandleFunc("/ws", handlers.HandleWebsocketConnection)
	router.HandleFunc("POST /message", handlers.HandlePostMessage)
	router.HandleFunc("DELETE /message", handlers.HandleDeleteMessage)

	handler := http.Handler(router)

	handler = otelhttp.NewHandler(handler, "maika-chat")

	httpServer := &http.Server{
		Addr:         ":8080",
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
		Handler:      handler,
	}

	if err := httpServer.ListenAndServe(); err != nil {
		log.Println(err.Error())
	}
}
