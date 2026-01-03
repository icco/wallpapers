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

	// indexTemplate is the parsed template for the homepage
	indexTemplate *template.Template
	// detailTemplate is the parsed template for the image detail page
	detailTemplate *template.Template
)

// PageData holds data passed to the index template
type PageData struct {
	Query  string
	Images []*db.Image
}

// DetailPageData holds data passed to the detail template
type DetailPageData struct {
	Image *db.Image
}

func main() {
	port := "8080"
	if fromEnv := os.Getenv("PORT"); fromEnv != "" {
		port = fromEnv
	}
	log.Infow("Starting up", "host", fmt.Sprintf("http://localhost:%s", port))

	// Parse templates from embedded files
	tmplContent, err := static.Assets.ReadFile("index.tmpl")
	if err != nil {
		log.Fatalw("failed to read index template", zap.Error(err))
	}
	indexTemplate, err = template.New("index").Parse(string(tmplContent))
	if err != nil {
		log.Fatalw("failed to parse index template", zap.Error(err))
	}

	detailContent, err := static.Assets.ReadFile("detail.tmpl")
	if err != nil {
		log.Fatalw("failed to read detail template", zap.Error(err))
	}
	detailTemplate, err = template.New("detail").Parse(string(detailContent))
	if err != nil {
		log.Fatalw("failed to parse detail template", zap.Error(err))
	}

	// Open database
	database, err = db.Open(db.DefaultDBPath())
	if err != nil {
		log.Warnw("could not open database, search will be unavailable", zap.Error(err))
	} else {
		defer func() {
			if cerr := database.Close(); cerr != nil {
				log.Errorw("failed to close database", zap.Error(cerr))
			}
		}()
		// Run data migrations
		if err := database.RunMigrations(); err != nil {
			log.Warnw("failed to run migrations", zap.Error(err))
		}
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

	// Serve static files (CSS, JS, etc.)
	r.Handle("/css/*", http.FileServer(http.FS(static.Assets)))
	r.Handle("/js/*", http.FileServer(http.FS(static.Assets)))
	r.Handle("/favicon.ico", http.FileServer(http.FS(static.Assets)))
	r.Handle("/robots.txt", http.FileServer(http.FS(static.Assets)))

	// Homepage with server-side filtering
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		query := r.URL.Query().Get("q")

		var images []*db.Image

		if query != "" && database != nil {
			// Search with query
			var err error
			images, err = database.Search(query)
			if err != nil {
				log.Errorw("error during search", zap.Error(err))
				http.Error(w, "Search error", http.StatusInternalServerError)
				return
			}
			// Add URLs to all images
			for _, img := range images {
				img.WithURLs()
			}
		} else {
			// Load all images from GCS
			gcsFiles, err := wallpapers.GetAll(ctx)
			if err != nil {
				log.Errorw("error during get all", zap.Error(err))
				http.Error(w, "Retrieval error", http.StatusInternalServerError)
				return
			}
			images = gcsFilesToImages(gcsFiles)
		}

		data := PageData{
			Query:  query,
			Images: images,
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := indexTemplate.Execute(w, data); err != nil {
			log.Errorw("error rendering template", zap.Error(err))
		}
	})

	r.Get("/all.json", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		gcsFiles, err := wallpapers.GetAll(ctx)
		if err != nil {
			log.Errorw("error during get all", zap.Error(err))
			if err := Renderer.JSON(w, 500, map[string]string{"error": "retrieval error"}); err != nil {
				log.Errorw("error during get all render", zap.Error(err))
			}
			return
		}

		// Convert GCS files to images with database metadata
		images := gcsFilesToImages(gcsFiles)

		if err := Renderer.JSON(w, http.StatusOK, images); err != nil {
			log.Errorw("error during get all success render", zap.Error(err))
		}
	})

	// Image detail page
	r.Get("/image/{filename}", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		filename := chi.URLParam(r, "filename")

		if filename == "" {
			http.Error(w, "Filename required", http.StatusBadRequest)
			return
		}

		var img *db.Image

		// First try to get from database (has full metadata)
		if database != nil {
			dbImg, err := database.GetByFilename(filename)
			if err != nil {
				log.Errorw("error fetching from database", zap.Error(err))
			}
			if dbImg != nil {
				img = dbImg.WithURLs()
			}
		}

		// If not in database, try to find in GCS
		if img == nil {
			gcsFiles, err := wallpapers.GetAll(ctx)
			if err != nil {
				log.Errorw("error during get all", zap.Error(err))
				http.Error(w, "Retrieval error", http.StatusInternalServerError)
				return
			}

			for _, f := range gcsFiles {
				if f.Name == filename {
					img = &db.Image{
						Filename:     f.Name,
						DateAdded:    f.Created,
						LastModified: f.Updated,
					}
					img.WithURLs()
					break
				}
			}
		}

		if img == nil {
			http.Error(w, "Image not found", http.StatusNotFound)
			return
		}

		data := DetailPageData{
			Image: img,
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := detailTemplate.Execute(w, data); err != nil {
			log.Errorw("error rendering detail template", zap.Error(err))
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

		images, err := database.Search(query)
		if err != nil {
			log.Errorw("error during search", zap.Error(err))
			if err := Renderer.JSON(w, 500, map[string]string{"error": "search error"}); err != nil {
				log.Errorw("error rendering search error", zap.Error(err))
			}
			return
		}

		// Add URLs to all images
		for _, img := range images {
			img.WithURLs()
		}

		if err := Renderer.JSON(w, http.StatusOK, images); err != nil {
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

// gcsFilesToImages converts GCS file listings to Image structs with database metadata.
func gcsFilesToImages(files []*wallpapers.File) []*db.Image {
	result := make([]*db.Image, 0, len(files))
	for _, f := range files {
		img := &db.Image{Filename: f.Name, DateAdded: f.Created, LastModified: f.Updated}
		if database != nil {
			dbImg, _ := database.GetByFilename(f.Name)
			img.MergeMetadata(dbImg)
		}
		result = append(result, img.WithURLs())
	}
	return result
}
