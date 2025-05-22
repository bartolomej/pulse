package server

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

var sequentialWhitespacePattern = regexp.MustCompile(`\s+`)
var whitespaceAtBeginningOfLinePattern = regexp.MustCompile(`(?m)^\s+`)

func percentChange(current, previous float64) float64 {
	if previous == 0 {
		if current == 0 {
			return 0 // 0% change if both are 0
		}
		return 100 // 100% increase if going from 0 to something
	}

	return (current/previous - 1) * 100
}

func isRunningInsideDockerContainer() bool {
	_, err := os.Stat("/.dockerenv")
	return err == nil
}

func prefixStringLines(prefix string, s string) string {
	lines := strings.Split(s, "\n")

	for i, line := range lines {
		lines[i] = prefix + line
	}

	return strings.Join(lines, "\n")
}

func titleToSlug(s string) string {
	s = strings.ToLower(s)
	s = sequentialWhitespacePattern.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")

	return s
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

func itemAtIndexOrDefault[T any](items []T, index int, def T) T {
	if index >= len(items) {
		return def
	}

	return items[index]
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
