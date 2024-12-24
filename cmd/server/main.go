package main

import (
	"fmt"
	"html/template"
	"net/http"
	"os"

	chi "github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/icco/gutil/etag"
	"github.com/icco/gutil/logging"
	"github.com/icco/wallpapers"
	"github.com/icco/wallpapers/cmd/server/static"
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
)

func main() {
	port := "8080"
	if fromEnv := os.Getenv("PORT"); fromEnv != "" {
		port = fromEnv
	}
	log.Infow("Starting up", "host", fmt.Sprintf("http://localhost:%s", port))

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

		if err := Renderer.JSON(w, http.StatusOK, images); err != nil {
			log.Errorw("error during get all success render", zap.Error(err))
		}
	})

	log.Fatal(http.ListenAndServe(":"+port, r))
}
