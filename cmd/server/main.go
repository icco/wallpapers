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

	// Renderer is a renderer for all occasions. These are our preferred default options.
	// See:
	//  - https://github.com/unrolled/render/blob/v1/README.md
	//  - https://godoc.org/gopkg.in/unrolled/render.v1
	Renderer = render.New(render.Options{
		Charset:                   "UTF-8",
		DisableHTTPErrorRendering: false,
		Extensions:                []string{".tmpl", ".html"},
		IndentJSON:                false,
		IndentXML:                 true,
		RequirePartials:           false,
		Funcs:                     []template.FuncMap{template.FuncMap{}},
	})
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

	r.Get("/all.json", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		images, err := wallpapers.GetAll(ctx)
		if err != nil {
			log.Errorw("error during get all", zap.Error(err))
			Renderer.JSON(w, 500, map[string]string{"error": "retrieval error"})
			return
		}

		Renderer.JSON(w, http.StatusOK, images)
	})

	log.Fatal(http.ListenAndServe(":"+port, r))
}
