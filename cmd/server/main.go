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

	indexTemplate       *template.Template
	detailTemplate      *template.Template
	resolutionsTemplate *template.Template
	colorsTemplate      *template.Template
	tagsTemplate        *template.Template

	// templateFuncs is registered on templates that use custom functions.
	templateFuncs = template.FuncMap{
		// tagRatio outputs count/maxCount for use as a CSS --ratio custom property.
		"tagRatio": func(count, maxCount int) string {
			if maxCount <= 0 {
				return "0"
			}
			return fmt.Sprintf("%.3f", float64(count)/float64(maxCount))
		},
	}
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

// ResolutionsPageData holds data passed to the resolutions template
type ResolutionsPageData struct {
	Resolutions []db.ResolutionEntry
}

// ColorsPageData holds data passed to the colors template
type ColorsPageData struct {
	Colors []db.ColorEntry
}

// TagsPageData holds data passed to the tags template
type TagsPageData struct {
	Tags     []db.TagEntry
	MaxCount int
}

// loadTemplate parses layout.tmpl and the named page file into a single template set.
// Pass funcs for pages that call custom template functions.
func loadTemplate(name string, funcs template.FuncMap) (*template.Template, error) {
	layoutContent, err := static.Assets.ReadFile("layout.tmpl")
	if err != nil {
		return nil, fmt.Errorf("read layout.tmpl: %w", err)
	}
	pageContent, err := static.Assets.ReadFile(name + ".tmpl")
	if err != nil {
		return nil, fmt.Errorf("read %s.tmpl: %w", name, err)
	}
	t := template.New("layout")
	if funcs != nil {
		t = t.Funcs(funcs)
	}
	if t, err = t.Parse(string(layoutContent)); err != nil {
		return nil, fmt.Errorf("parse layout.tmpl: %w", err)
	}
	if t, err = t.Parse(string(pageContent)); err != nil {
		return nil, fmt.Errorf("parse %s.tmpl: %w", name, err)
	}
	return t, nil
}

func main() {
	port := "8080"
	if fromEnv := os.Getenv("PORT"); fromEnv != "" {
		port = fromEnv
	}
	log.Infow("Starting up", "host", fmt.Sprintf("http://localhost:%s", port))

	var err error
	if indexTemplate, err = loadTemplate("index", nil); err != nil {
		log.Fatalw("failed to load index template", zap.Error(err))
	}
	if detailTemplate, err = loadTemplate("detail", nil); err != nil {
		log.Fatalw("failed to load detail template", zap.Error(err))
	}
	if resolutionsTemplate, err = loadTemplate("resolutions", nil); err != nil {
		log.Fatalw("failed to load resolutions template", zap.Error(err))
	}
	if colorsTemplate, err = loadTemplate("colors", nil); err != nil {
		log.Fatalw("failed to load colors template", zap.Error(err))
	}
	if tagsTemplate, err = loadTemplate("tags", templateFuncs); err != nil {
		log.Fatalw("failed to load tags template", zap.Error(err))
	}

	database, err = db.Open(db.DefaultDBPath())
	if err != nil {
		log.Warnw("could not open database, search will be unavailable", zap.Error(err))
	} else {
		defer func() {
			if cerr := database.Close(); cerr != nil {
				log.Errorw("failed to close database", zap.Error(cerr))
			}
		}()
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

	r.Handle("/css/*", http.FileServer(http.FS(static.Assets)))
	r.Handle("/js/*", http.FileServer(http.FS(static.Assets)))
	r.Handle("/favicon.ico", http.FileServer(http.FS(static.Assets)))
	r.Handle("/robots.txt", http.FileServer(http.FS(static.Assets)))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("q")

		if database == nil {
			log.Errorw("database not available")
			http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
			return
		}

		var images []*db.Image
		var err error

		if query != "" {
			images, err = database.Search(query)
		} else {
			images, err = database.GetAll()
		}
		if err != nil {
			log.Errorw("error fetching images", zap.Error(err))
			http.Error(w, "Retrieval error", http.StatusInternalServerError)
			return
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
		if database == nil {
			log.Errorw("database not available")
			if err := Renderer.JSON(w, 503, map[string]string{"error": "service unavailable"}); err != nil {
				log.Errorw("error rendering unavailable", zap.Error(err))
			}
			return
		}

		images, err := database.GetAll()
		if err != nil {
			log.Errorw("error during get all", zap.Error(err))
			if err := Renderer.JSON(w, 500, map[string]string{"error": "retrieval error"}); err != nil {
				log.Errorw("error during get all render", zap.Error(err))
			}
			return
		}

		if err := Renderer.JSON(w, http.StatusOK, images); err != nil {
			log.Errorw("error during get all success render", zap.Error(err))
		}
	})

	r.Get("/image/{filename}", func(w http.ResponseWriter, r *http.Request) {
		filename := chi.URLParam(r, "filename")

		if filename == "" {
			http.Error(w, "Filename required", http.StatusBadRequest)
			return
		}

		if database == nil {
			log.Errorw("database not available")
			http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
			return
		}

		img, err := database.GetByFilename(filename)
		if err != nil {
			log.Errorw("error fetching from database", zap.Error(err))
			http.Error(w, "Retrieval error", http.StatusInternalServerError)
			return
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

	r.Get("/resolutions", func(w http.ResponseWriter, r *http.Request) {
		if database == nil {
			log.Errorw("database not available")
			http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
			return
		}
		resolutions, err := database.GetResolutions()
		if err != nil {
			log.Errorw("error fetching resolutions", zap.Error(err))
			http.Error(w, "Retrieval error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := resolutionsTemplate.Execute(w, ResolutionsPageData{Resolutions: resolutions}); err != nil {
			log.Errorw("error rendering resolutions template", zap.Error(err))
		}
	})

	r.Get("/colors", func(w http.ResponseWriter, r *http.Request) {
		if database == nil {
			log.Errorw("database not available")
			http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
			return
		}
		colors, err := database.GetColors()
		if err != nil {
			log.Errorw("error fetching colors", zap.Error(err))
			http.Error(w, "Retrieval error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := colorsTemplate.Execute(w, ColorsPageData{Colors: colors}); err != nil {
			log.Errorw("error rendering colors template", zap.Error(err))
		}
	})

	r.Get("/tags", func(w http.ResponseWriter, r *http.Request) {
		if database == nil {
			log.Errorw("database not available")
			http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
			return
		}
		tags, err := database.GetTags()
		if err != nil {
			log.Errorw("error fetching tags", zap.Error(err))
			http.Error(w, "Retrieval error", http.StatusInternalServerError)
			return
		}
		maxCount := 0
		if len(tags) > 0 {
			maxCount = tags[0].Count
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tagsTemplate.Execute(w, TagsPageData{Tags: tags, MaxCount: maxCount}); err != nil {
			log.Errorw("error rendering tags template", zap.Error(err))
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
