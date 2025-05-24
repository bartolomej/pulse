package server

import (
	"bytes"
	"fmt"
	"github.com/glanceapp/glance/pkg/sources/common"
	"github.com/glanceapp/glance/pkg/widgets"
	"github.com/glanceapp/glance/web"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var (
	pageTemplate        = web.MustParseTemplate("page.html", "document.html", "footer.html")
	pageContentTemplate = web.MustParseTemplate("page-content.html")
	manifestTemplate    = web.MustParseTemplate("manifest.json")
)

const STATIC_ASSETS_CACHE_DURATION = 24 * time.Hour

type Application struct {
	Version        string
	CreatedAt      time.Time
	Config         config
	parsedManifest []byte
}

func NewApplication() (*Application, error) {
	app := &Application{
		Version:   common.BuildVersion,
		CreatedAt: time.Now(),
	}

	if app.Config.FaviconURL == "" {
		app.Config.FaviconURL = app.StaticAssetPath("favicon.svg")
	} else {
		app.Config.FaviconURL = app.resolveUserDefinedAssetPath(app.Config.FaviconURL)
	}

	if strings.HasSuffix(app.Config.FaviconType, ".svg") {
		app.Config.FaviconType = "image/svg+xml"
	} else {
		app.Config.FaviconType = "image/png"
	}

	manifest, err := executeTemplateToString(manifestTemplate, templateData{App: app})
	if err != nil {
		return nil, fmt.Errorf("parsing manifest.json: %v", err)
	}
	app.parsedManifest = []byte(manifest)

	return app, nil
}

func (a *Application) resolveUserDefinedAssetPath(path string) string {
	if strings.HasPrefix(path, "/assets/") {
		return a.Config.BaseURL + path
	}

	return path
}

type templateRequestData struct {
	Theme  *widgets.Theme
	Filter string
}

type templateData struct {
	App     *Application
	Page    *widgets.Page
	Request templateRequestData
}

func (a *Application) populateTemplateRequestData(data *templateRequestData, r *http.Request) {
	// TODO(pulse): Update theme retrieval
	//theme := &a.Config.Theme.Theme
	//
	//selectedTheme, err := r.Cookie("theme")
	//if err == nil {
	//	preset, exists := a.Config.Theme.Presets.Get(selectedTheme.Value)
	//	if exists {
	//		theme = preset
	//	}
	//}
	//
	//data.Theme = theme
	//data.Filter = r.URL.Query().Get("filter")
}

func (a *Application) handlePageRequest(w http.ResponseWriter, r *http.Request) {
	data := templateData{
		//Page: page,
		App: a,
	}
	a.populateTemplateRequestData(&data.Request, r)

	var responseBytes bytes.Buffer
	err := pageTemplate.Execute(&responseBytes, data)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	w.Write(responseBytes.Bytes())
}

func (a *Application) handlePageContentRequest(w http.ResponseWriter, r *http.Request) {
	pageData := templateData{
		//Page: page,
	}

	a.populateTemplateRequestData(&pageData.Request, r)
	var responseBytes bytes.Buffer
	err := pageContentTemplate.Execute(&responseBytes, pageData)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
}

func (a *Application) addressOfRequest(r *http.Request) string {
	remoteAddrWithoutPort := func() string {
		for i := len(r.RemoteAddr) - 1; i >= 0; i-- {
			if r.RemoteAddr[i] == ':' {
				return r.RemoteAddr[:i]
			}
		}

		return r.RemoteAddr
	}

	if !a.Config.Proxied {
		return remoteAddrWithoutPort()
	}

	// This should probably be configurable or look for multiple headers, not just this one
	forwardedFor := r.Header.Get("X-Forwarded-For")
	if forwardedFor == "" {
		return remoteAddrWithoutPort()
	}

	ips := strings.Split(forwardedFor, ",")
	if len(ips) == 0 || ips[0] == "" {
		return remoteAddrWithoutPort()
	}

	return ips[0]
}

func (a *Application) handleNotFound(w http.ResponseWriter, _ *http.Request) {
	// TODO: add proper not found page
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("Page not found"))
}

func (a *Application) StaticAssetPath(asset string) string {
	return a.Config.BaseURL + "/static/" + web.StaticFSHash + "/" + asset
}

func (a *Application) VersionedAssetPath(asset string) string {
	return a.Config.BaseURL + asset +
		"?v=" + strconv.FormatInt(a.CreatedAt.Unix(), 10)
}

func (a *Application) Server() (func() error, func() error) {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /{$}", a.handlePageRequest)
	mux.HandleFunc("GET /{page}", a.handlePageRequest)

	mux.HandleFunc("GET /api/pages/{page}/content/{$}", a.handlePageContentRequest)
	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mux.Handle(
		fmt.Sprintf("GET /static/%s/{path...}", web.StaticFSHash),
		http.StripPrefix(
			"/static/"+web.StaticFSHash,
			fileServerWithCache(http.FS(web.StaticFS), STATIC_ASSETS_CACHE_DURATION),
		),
	)

	assetCacheControlValue := fmt.Sprintf(
		"public, max-age=%d",
		int(STATIC_ASSETS_CACHE_DURATION.Seconds()),
	)

	mux.HandleFunc(fmt.Sprintf("GET /static/%s/css/bundle.css", web.StaticFSHash), func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Cache-Control", assetCacheControlValue)
		w.Header().Add("Content-Type", "text/css; charset=utf-8")
		w.Write(web.BundledCSSContents)
	})

	mux.HandleFunc("GET /manifest.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Cache-Control", assetCacheControlValue)
		w.Header().Add("Content-Type", "Application/json")
		w.Write(a.parsedManifest)
	})

	var absAssetsPath string
	if a.Config.AssetsPath != "" {
		absAssetsPath, _ = filepath.Abs(a.Config.AssetsPath)
		assetsFS := fileServerWithCache(http.Dir(a.Config.AssetsPath), 2*time.Hour)
		mux.Handle("/assets/{path...}", http.StripPrefix("/assets/", assetsFS))
	}

	server := http.Server{
		Addr:    fmt.Sprintf("%s:%d", a.Config.Host, a.Config.Port),
		Handler: mux,
	}

	start := func() error {
		log.Printf("Starting server on %s:%d (base-url: \"%s\", assets-path: \"%s\")\n",
			a.Config.Host,
			a.Config.Port,
			a.Config.BaseURL,
			absAssetsPath,
		)

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return err
		}

		return nil
	}

	stop := func() error {
		return server.Close()
	}

	return start, stop
}
