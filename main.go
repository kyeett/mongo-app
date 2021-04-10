package main

import (
	"context"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	uri := os.Getenv("MONGO_URI")
	if uri == "" {
		log.Fatal("please specify MONGO_URI")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "3333"
	}
	log.Println("using PORT=", port)

	// Setup database connections
	client, err := newConnectedMongoClient(uri)
	if err != nil {
		log.Fatal(err)
	}

	defer func() {
		_ = client.Disconnect(context.Background())
	}()

	r := chi.NewRouter()

	// A good base middleware stack
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Use(middleware.Timeout(10 * time.Second))

	r.Get("/healthcheck", healthCheckHandler(client))
	r.Get("/info", infoHandler)

	if err := http.ListenAndServe(":" + port, r); err != nil {
		log.Fatal(err)
	}
}

func newConnectedMongoClient(uri string) (*mongo.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("failed to start connection to mongo db: %w", err)
	}

	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		return nil, fmt.Errorf("failed to ping mongo db: %w", err)
	}
	return client, nil
}

func healthCheckHandler(client *mongo.Client) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		err := client.Ping(r.Context(), readpref.Primary())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("failed to connect to db"))
			return
		}

		w.Write([]byte("OK!"))
	}
}

func infoHandler(w http.ResponseWriter, r *http.Request) {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "missing"
	}

	resp := fmt.Sprintf(`
hostname=  %s
host    =  %s
url     =  %s`,
		hostname, r.Host, r.URL.String())

	w.Write([]byte(resp))
}
