package prefixproxy

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/traefik/traefik/v3/pkg/middlewares"
	"go.opentelemetry.io/otel/trace"
)

const (
	typeName = "PrefixProxy"
)

// Config holds the middleware configuration.
type Config struct {
	Prefix string `json:"prefix,omitempty"`
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{
		Prefix: "",
	}
}

// PrefixProxy is a middleware that removes a specified prefix from the request path
// and adds it back to the response.
type prefixProxy struct {
	next   http.Handler
	prefix string
	name   string
}

// New creates a new PrefixProxy middleware.
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	middlewares.GetLogger(ctx, name, typeName).Debug().Msg("Creating middleware")

	if len(config.Prefix) == 0 {
		return nil, errors.New("prefix cannot be empty")
	}

	return &prefixProxy{
		next:   next,
		prefix: strings.Trim(config.Prefix, "/"),
		name:   name,
	}, nil
}

func (p *prefixProxy) GetTracingInformation() (string, string, trace.SpanKind) {
	return p.name, typeName, trace.SpanKindInternal
}

func (p *prefixProxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	logger := middlewares.GetLogger(req.Context(), p.name, typeName)

	var prefixRemoved bool
	oldPath := req.URL.Path
	if strings.HasPrefix(req.URL.Path, "/"+p.prefix) {
		req.URL.Path = strings.TrimPrefix(req.URL.Path, "/"+p.prefix)
		if req.URL.Path == "" {
			req.URL.Path = "/"
		}
		prefixRemoved = true
		logger.Debug().Msgf("URL.Path is now %s (was %s).", req.URL.Path, oldPath)
	}

	if req.URL.RawPath != "" {
		oldRawPath := req.URL.RawPath
		if strings.HasPrefix(req.URL.RawPath, "/"+p.prefix) {
			req.URL.RawPath = strings.TrimPrefix(req.URL.RawPath, "/"+p.prefix)
			if req.URL.RawPath == "" {
				req.URL.RawPath = "/"
			}
			prefixRemoved = true
			logger.Debug().Msgf("URL.RawPath is now %s (was %s).", req.URL.RawPath, oldRawPath)
		}
	}

	req.RequestURI = req.URL.RequestURI()

	crw := &customResponseWriter{
		ResponseWriter: rw,
		prefix:         p.prefix,
		prefixRemoved:  prefixRemoved,
	}

	p.next.ServeHTTP(crw, req)
}

type customResponseWriter struct {
	http.ResponseWriter
	prefix        string
	prefixRemoved bool
}

func (crw *customResponseWriter) WriteHeader(statusCode int) {
	if location := crw.Header().Get("Location"); location != "" && crw.prefixRemoved {
		if !strings.HasPrefix(location, "/"+crw.prefix) {
			crw.Header().Set("Location", "/"+crw.prefix+location)
		}
	}
	crw.ResponseWriter.WriteHeader(statusCode)
}
