package main

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"time"

	chi "github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/icco/gutil/etag"
	"github.com/icco/gutil/logging"
	"github.com/icco/wallpapers"
	"github.com/icco/wallpapers/cmd/server/static"
	"github.com/icco/wallpapers/db"
	"github.com/unrolled/render"
	"github.com/unrolled/secure"
	"go.uber.org/zap"
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

	database *db.DB
)

// EnrichedFile extends the File struct with database metadata.
type EnrichedFile struct {
	*wallpapers.File
	Width        int      `json:"width,omitempty"`
	Height       int      `json:"height,omitempty"`
	PixelDensity float64  `json:"pixel_density,omitempty"`
	FileFormat   string   `json:"file_format,omitempty"`
	Color1       string   `json:"color1,omitempty"`
	Color2       string   `json:"color2,omitempty"`
	Color3       string   `json:"color3,omitempty"`
	Words        []string `json:"words,omitempty"`
}

func main() {
	port := "8080"
	if fromEnv := os.Getenv("PORT"); fromEnv != "" {
		port = fromEnv
	}
	log.Infow("Starting up", "host", fmt.Sprintf("http://localhost:%s", port))

	// Open database
	var err error
	database, err = db.Open(db.DefaultDBPath())
	if err != nil {
		log.Warnw("could not open database, search will be unavailable", zap.Error(err))
	} else {
		defer database.Close()
	}

	secureMiddleware := secure.New(secure.Options{
		SSLRedirect:        false,
		SSLProxyHeaders:    map[string]string{"X-Forwarded-Proto": "https"},
		FrameDeny:          true,
		ContentTypeNosniff: true,
		BrowserXssFilter:   true,
		ReferrerPolicy:     "no-referrer",
		FeaturePolicy:      "geolocation 'none'; midi 'none'; sync-xhr 'none'; microphone 'none'; camera 'none'; magnetometer 'none'; gyroscope 'none'; fullscreen 'none'; payment 'none'; usb 'none'",
	})

	r := chi.NewRouter()
	r.Use(etag.Handler(false))
	r.Use(middleware.RealIP)
	r.Use(logging.Middleware(log.Desugar(), project))
	r.Use(secureMiddleware.Handler)

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

	r.Use(func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("report-to", `{"group":"default","max_age":10886400,"endpoints":[{"url":"https://reportd.natwelch.com/report/wallpapers"}]}`)
			w.Header().Set("reporting-endpoints", `default="https://reportd.natwelch.com/reporting/wallpapers"`)

			h.ServeHTTP(w, r)
		})
	})

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte("hi.")); err != nil {
			log.Errorw("error writing healthz", zap.Error(err))
		}
	})

	r.Mount("/", http.FileServer(http.FS(static.Assets)))

	r.Get("/all.json", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		images, err := wallpapers.GetAll(ctx)
		if err != nil {
			log.Errorw("error during get all", zap.Error(err))
			if err := Renderer.JSON(w, 500, map[string]string{"error": "retrieval error"}); err != nil {
				log.Errorw("error during get all render", zap.Error(err))
			}
			return
		}

		// Enrich with database metadata
		enriched := enrichImages(images)

		if err := Renderer.JSON(w, http.StatusOK, enriched); err != nil {
			log.Errorw("error during get all success render", zap.Error(err))
		}
	})

	r.Get("/search", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("q")

		if database == nil {
			log.Errorw("search unavailable, database not connected")
			if err := Renderer.JSON(w, 503, map[string]string{"error": "search unavailable"}); err != nil {
				log.Errorw("error rendering search error", zap.Error(err))
			}
			return
		}

		dbImages, err := database.Search(query)
		if err != nil {
			log.Errorw("error during search", zap.Error(err))
			if err := Renderer.JSON(w, 500, map[string]string{"error": "search error"}); err != nil {
				log.Errorw("error rendering search error", zap.Error(err))
			}
			return
		}

		// Convert database images to enriched files
		results := make([]*EnrichedFile, 0, len(dbImages))
		for _, dbImg := range dbImages {
			file := &wallpapers.File{
				Name:         dbImg.Filename,
				ThumbnailURL: wallpapers.ThumbURL(dbImg.Filename),
				FullRezURL:   wallpapers.FullRezURL(dbImg.Filename),
				Created:      dbImg.DateAdded,
				Updated:      dbImg.LastModified,
			}

			enriched := &EnrichedFile{
				File:         file,
				Width:        dbImg.Width,
				Height:       dbImg.Height,
				PixelDensity: dbImg.PixelDensity,
				FileFormat:   dbImg.FileFormat,
				Color1:       dbImg.Color1,
				Color2:       dbImg.Color2,
				Color3:       dbImg.Color3,
				Words:        dbImg.Words,
			}
			results = append(results, enriched)
		}

		if err := Renderer.JSON(w, http.StatusOK, results); err != nil {
			log.Errorw("error rendering search results", zap.Error(err))
		}
	})

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  10 * time.Second,
	}

	log.Fatal(srv.ListenAndServe())
}

// enrichImages adds database metadata to GCS file listings.
func enrichImages(files []*wallpapers.File) []*EnrichedFile {
	result := make([]*EnrichedFile, 0, len(files))

	for _, f := range files {
		enriched := &EnrichedFile{File: f}

		if database != nil {
			if dbImg, err := database.GetByFilename(f.Name); err == nil && dbImg != nil {
				enriched.Width = dbImg.Width
				enriched.Height = dbImg.Height
				enriched.PixelDensity = dbImg.PixelDensity
				enriched.FileFormat = dbImg.FileFormat
				enriched.Color1 = dbImg.Color1
				enriched.Color2 = dbImg.Color2
				enriched.Color3 = dbImg.Color3
				enriched.Words = dbImg.Words
			}
		}

		result = append(result, enriched)
	}

	return result
}
