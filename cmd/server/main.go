package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/icco/gutil/etag"
	"github.com/icco/gutil/logging"
	"github.com/icco/gutil/otel"
	"github.com/icco/wallpapers/cmd/server/static"
)

const (
	service = "walls"
	project = "icco-cloud"
)

var (
	log = logging.Must(logging.NewLogger(service))
)

func main() {
	ctx := context.Background()
	port := "8080"
	if fromEnv := os.Getenv("PORT"); fromEnv != "" {
		port = fromEnv
	}
	log.Infow("Starting up", "host", fmt.Sprintf("http://localhost:%s", port))

	if err := otel.Init(ctx, log, project, service); err != nil {
		log.Errorw("could not init opentelemetry", zap.Error(err))
	}

	r := chi.NewRouter()
	r.Use(etag.Handler(false))
	r.Use(middleware.RealIP)
	r.Use(logging.Middleware(log.Desugar(), project))
	r.Use(otel.Middleware)

	crs := cors.New(cors.Options{
		AllowCredentials:   true,
		OptionsPassthrough: false,
		AllowedOrigins:     []string{"*"},
		AllowedMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:     []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:     []string{"Link"},
		MaxAge:             300, // Maximum value not ignored by any of major browsers
	})
	r.Use(crs.Handler)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hi."))
	})

	r.Mount("/", http.FileServer(http.FS(static.Assets)))

	log.Fatal(http.ListenAndServe(":"+port, r))
}
