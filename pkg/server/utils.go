package server

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"time"
)

func fileServerWithCache(fs http.FileSystem, cacheDuration time.Duration) http.Handler {
	server := http.FileServer(fs)
	cacheControlValue := fmt.Sprintf("public, max-age=%d", int(cacheDuration.Seconds()))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO: fix always setting cache control even if the file doesn't exist
		w.Header().Set("Cache-Control", cacheControlValue)
		server.ServeHTTP(w, r)
	})
}

func ternary[T any](condition bool, a, b T) T {
	if condition {
		return a
	}

	return b
}

// Having compile time errors about unused variables is cool and all, but I don't want to
// have to constantly comment out my code while I'm working on it and testing things out
func ItsUsedTrustMeBro(...any) {}

func executeTemplateToString(t *template.Template, data any) (string, error) {
	var b bytes.Buffer
	err := t.Execute(&b, data)
	if err != nil {
		return "", fmt.Errorf("executing template: %w", err)
	}

	return b.String(), nil
}
