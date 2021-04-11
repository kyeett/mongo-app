package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"log"
	"net/http"
	"os"
	"strings"
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
	log.Println("using PORT =", port)

	// Setup database connections
	client, err := NewConnectedMongoClient(uri)
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
	r.Get("/data/*", dataGetHandler(client))
	r.Post("/data/*", dataPostHandler(client))

	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal(err)
	}
}

type service struct {
	client *mongo.Client
}

func (s *service) GetData(ctx context.Context, object string) (*bson.D, error) {
	collection := s.client.Database("app-db").Collection(object)

	res := collection.FindOne(ctx, bson.D{}, options.FindOne().SetSort(bson.M{"_id": -1}))
	if err := res.Err(); err != nil {
		return nil, err
	}

	var result bson.D
	err := res.Decode(&result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (s *service) InsertData(ctx context.Context, object string, d bson.D) error {
	collection := s.client.Database("app-db").Collection(object)
	_, err := collection.InsertOne(ctx, d)
	if err != nil {
		return err
	}

	return nil
}

func dataGetHandler(client *mongo.Client) http.HandlerFunc {
	s := service{client: client}
	return func(w http.ResponseWriter, r *http.Request) {
		object := strings.TrimLeft(r.URL.Path, "/data/")

		v, err := s.GetData(r.Context(), object)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("failed to connect to db: " + err.Error()))
			return
		}

		b, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("failed to marshal data: " + err.Error()))
			return
		}

		w.Write(b)
	}
}

func dataPostHandler(client *mongo.Client) http.HandlerFunc {
	s := service{client: client}
	return func(w http.ResponseWriter, r *http.Request) {
		object := strings.TrimLeft(r.URL.Path, "/data/")

		if err := r.ParseForm(); err != nil {
			w.Write([]byte("failed to handle request: " + err.Error()))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		var d bson.D
		for i, v := range r.Form {
			d = append(d, bson.E{i, v})
		}

		if err := s.InsertData(r.Context(), object, d); err != nil {
			w.Write([]byte("failed to handle request: " + err.Error()))
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}

func NewConnectedMongoClient(uri string) (*mongo.Client, error) {
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
