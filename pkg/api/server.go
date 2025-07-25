package api

import (
	"bytes"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/glanceapp/glance/pkg/sources/activities/types"
	"github.com/glanceapp/glance/pkg/sources/nlp"
	httpswagger "github.com/swaggo/http-swagger"

	"github.com/glanceapp/glance/pkg/storage/postgres"
	"github.com/tmc/langchaingo/llms/openai"

	"github.com/glanceapp/glance/pkg/sources"
	"github.com/glanceapp/glance/pkg/widgets"
	"github.com/glanceapp/glance/web"
	"github.com/rs/zerolog"
)

//go:embed openapi.yaml
var openapiSpecYaml string

const StaticAssetsCacheDuration = 24 * time.Hour

var (
	pageTemplate        = web.MustParseTemplate("page.html", "document.html", "footer.html", "page-content.html")
	pageContentTemplate = web.MustParseTemplate("page-content.html")
)

type Server struct {
	registry  *sources.Registry
	logger    *zerolog.Logger
	createdAt time.Time
	config    *Config
	http      http.Server
}

var _ ServerInterface = (*Server)(nil)

func NewServer(logger *zerolog.Logger, cfg *Config, db *postgres.DB) (*Server, error) {
	summarizerModel, err := openai.New(
		openai.WithModel("gpt-4o-mini"),
	)
	if err != nil {
		return nil, fmt.Errorf("create summarizer model: %w", err)
	}

	embedderModel, err := openai.New(
		openai.WithEmbeddingModel("text-embedding-3-large"),
	)
	if err != nil {
		return nil, fmt.Errorf("create embedder model: %w", err)
	}

	registry := sources.NewRegistry(
		logger,
		nlp.NewSummarizer(summarizerModel),
		nlp.NewEmbedder(embedderModel),
		postgres.NewActivityRepository(db),
		postgres.NewSourceRepository(db),
	)

	mux := http.NewServeMux()

	server := &Server{
		createdAt: time.Now(),
		logger:    logger,
		config:    cfg,
		registry:  registry,
		http: http.Server{
			Addr:    fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
			Handler: corsMiddleware(mux),
		},
	}

	HandlerFromMux(server, mux)
	server.registerFileHandlers(mux)
	server.registerApiDocsHandlers(mux)

	return server, nil
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) registerApiDocsHandlers(mux *http.ServeMux) {
	mux.Handle("/docs/", httpswagger.Handler(
		httpswagger.URL("/docs/openapi.yaml"),
	))
	mux.HandleFunc("/docs/openapi.yaml", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/x-yaml")

		_, err := w.Write([]byte(openapiSpecYaml))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			s.logger.Error().Err(err).Msg("response write error")
		}
	})
}

func (s *Server) registerFileHandlers(mux *http.ServeMux) {
	mux.Handle(
		fmt.Sprintf("GET /static/%s/{path...}", web.StaticFSHash),
		http.StripPrefix(
			"/static/"+web.StaticFSHash,
			fileServerWithCache(http.FS(web.StaticFS), StaticAssetsCacheDuration),
		),
	)

	assetCacheControlValue := fmt.Sprintf(
		"public, max-age=%d",
		int(StaticAssetsCacheDuration.Seconds()),
	)

	mux.HandleFunc(fmt.Sprintf("GET /static/%s/css/bundle.css", web.StaticFSHash), func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Cache-Control", assetCacheControlValue)
		w.Header().Add("Content-Type", "text/css; charset=utf-8")
		w.Write(web.BundledCSSContents)
	})

	// TODO(pulse): Serve manifest at GET /manifest.json

	if s.config.AssetsPath != "" {
		assetsFS := fileServerWithCache(http.Dir(s.config.AssetsPath), 2*time.Hour)
		mux.Handle("/assets/{path...}", http.StripPrefix("/assets/", assetsFS))
	}
}

func (s *Server) Start() error {
	if err := s.http.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func (s *Server) Stop() error {
	return s.http.Close()
}

type templateData struct {
	Config         *Config
	Page           *widgets.Page
	Theme          widgets.Theme
	ThemePresets   []widgets.Theme
	SourceRegistry *sources.Registry
}

func (s *Server) GetPage(w http.ResponseWriter, r *http.Request, params GetPageParams) {
	configJson, err := base64.StdEncoding.DecodeString(params.Config)
	if err != nil {
		s.badRequest(w, err, "decode config")
		return
	}

	page, err := widgets.NewPageFromJSON(configJson)
	if err != nil {
		s.badRequest(w, err, "deserialize page")
	}

	themePresets := widgets.DefaultThemePresets()
	data := templateData{
		Page:           page,
		Config:         s.config,
		Theme:          themePresets[0],
		ThemePresets:   themePresets,
		SourceRegistry: s.registry,
	}

	var responseBytes bytes.Buffer
	err = pageTemplate.Execute(&responseBytes, data)
	if err != nil {
		s.internalError(w, err, "execute template")
		return
	}

	_, err = w.Write(responseBytes.Bytes())
	if err != nil {
		s.logger.Err(err).Msg("write response")
	}
}

func (s *Server) ListAllActivities(w http.ResponseWriter, r *http.Request) {
	out, err := s.registry.Activities()
	if err != nil {
		s.internalError(w, err, "list activities")
		return
	}

	s.serializeRes(w, serializeActivities(out))
}

func (s *Server) ListSources(w http.ResponseWriter, r *http.Request) {
	out, err := s.registry.Sources()
	if err != nil {
		s.internalError(w, err, "list sources")
		return
	}

	s.serializeRes(w, serializeSources(out))
}

func (s *Server) CreateSource(w http.ResponseWriter, r *http.Request) {
	var req CreateSourceRequest
	err := deserializeReq(r, &req)
	if err != nil {
		s.badRequest(w, err, "deserialize request")
		return
	}

	out, err := deserializeCreateSourceRequest(req)
	if err != nil {
		s.badRequest(w, err, "deserialize request")
		return
	}

	err = s.registry.Add(out)
	if err != nil {
		s.internalError(w, err, "add source")
		return
	}

	s.serializeRes(w, deserializeSource(out))
}

func (s *Server) DeleteSource(w http.ResponseWriter, r *http.Request, uid string) {
	err := s.registry.Remove(uid)
	if err != nil {
		s.internalError(w, err, "remove source")
		return
	}

	s.serializeRes(w, nil)
}

func (s *Server) GetSource(w http.ResponseWriter, r *http.Request, uid string) {
	out, err := s.registry.Source(uid)
	if err != nil {
		s.internalError(w, err, "remove source")
		return
	}

	s.serializeRes(w, deserializeSource(out))
}

func (s *Server) SearchActivities(w http.ResponseWriter, r *http.Request, params SearchActivitiesParams) {
	var sourceUIDs []string
	if params.Sources != nil {
		sourceUIDs = strings.Split(*params.Sources, ",")
	}

	var minSimilarity float32
	if params.MinSimilarity != nil {
		minSimilarity = *params.MinSimilarity
	}

	var query string
	if params.Query != nil {
		query = *params.Query
	}

	limit := 20
	if params.Limit != nil {
		if *params.Limit < 1 || *params.Limit > 100 {
			s.badRequest(w, fmt.Errorf("limit must be between 1 and 100"), "validate limit")
			return
		}
		limit = *params.Limit
	}

	sortBy, err := deserializeSortBy(params.SortBy)
	if err != nil {
		s.badRequest(w, err, "deserialize sort by")
		return
	}

	results, err := s.registry.Search(r.Context(), query, sourceUIDs, minSimilarity, limit, sortBy)
	if err != nil {
		s.internalError(w, err, "search activities")
		return
	}

	s.serializeRes(w, serializeActivities(results))
}

func deserializeReq[Req any](r *http.Request, req *Req) error {
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		return fmt.Errorf("unsupported content type: %s", contentType)
	}

	reqBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("read request body: %w", err)
	}

	err = json.Unmarshal(reqBytes, req)
	if err != nil {
		return fmt.Errorf("deserialize request body: %w", err)
	}

	return nil
}

func (s *Server) serializeRes(w http.ResponseWriter, res any) {
	w.Header().Add("Content-Type", "application/json")

	if res == nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	err := json.NewEncoder(w).Encode(res)
	if err != nil {
		s.internalError(w, err, "serialize response")
	}
}

func (s *Server) internalError(w http.ResponseWriter, err error, msg string) {
	s.logger.Err(err).Msg(msg)
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

func (s *Server) badRequest(w http.ResponseWriter, err error, msg string) {
	s.logger.Err(err).Msg(msg)
	http.Error(w, err.Error(), http.StatusBadRequest)
}

func (s *Server) notFound(w http.ResponseWriter, err error, msg string) {
	s.logger.Err(err).Msg(msg)
	http.Error(w, err.Error(), http.StatusNotFound)
}

func deserializeCreateSourceRequest(req CreateSourceRequest) (sources.Source, error) {
	source, err := sources.NewSource(req.Type)
	if err != nil {
		return nil, fmt.Errorf("create source: %w", err)
	}

	configBytes, err := json.Marshal(req.Config)
	if err != nil {
		return nil, fmt.Errorf("marshal config: %w", err)
	}

	if err := json.Unmarshal(configBytes, source); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	return source, nil
}

func serializeActivities(in []*types.DecoratedActivity) []Activity {
	out := make([]Activity, 0, len(in))

	for _, e := range in {
		out = append(out, serializeActivity(e))
	}

	return out
}

func serializeActivity(in *types.DecoratedActivity) Activity {
	return Activity{
		Body:         in.Body(),
		CreatedAt:    in.CreatedAt(),
		ImageUrl:     in.ImageURL(),
		FullSummary:  in.Summary.FullSummary,
		ShortSummary: in.Summary.ShortSummary,
		SourceUid:    in.SourceUID(),
		Title:        in.Title(),
		Uid:          in.UID(),
		Url:          in.URL(),
		Similarity:   &in.Similarity,
	}
}

func serializeSources(in []sources.Source) []Source {
	out := make([]Source, 0, len(in))

	for _, e := range in {
		out = append(out, deserializeSource(e))
	}

	return out

}

func deserializeSource(in sources.Source) Source {
	return Source{
		Uid:  in.UID(),
		Url:  in.URL(),
		Name: in.Name(),
	}
}

func deserializeSortBy(in *SearchActivitiesParamsSortBy) (types.SortBy, error) {
	if in == nil {
		return types.SortByDate, nil
	}

	switch *in {
	case CreatedDate:
		return types.SortByDate, nil
	case Similarity:
		return types.SortBySimilarity, nil
	}

	return "", fmt.Errorf("unknown sort by: %s", *in)
}

func fileServerWithCache(fs http.FileSystem, cacheDuration time.Duration) http.Handler {
	server := http.FileServer(fs)
	cacheControlValue := fmt.Sprintf("public, max-age=%d", int(cacheDuration.Seconds()))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO: fix always setting cache control even if the file doesn't exist
		w.Header().Set("Cache-Control", cacheControlValue)
		server.ServeHTTP(w, r)
	})
}

func renderTemplate(t *template.Template, data any) (string, error) {
	var b bytes.Buffer
	err := t.Execute(&b, data)
	if err != nil {
		return "", fmt.Errorf("executing template: %w", err)
	}

	return b.String(), nil
}
