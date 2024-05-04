package main

import (
	"errors"
	"fmt"
	"mime"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write(knownPages.Home)
	})

	mux.HandleFunc("GET /yoke-website/{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write(knownPages.Home)
	})

	mux.HandleFunc("GET /", func() http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			content, err := content.ReadFile(path.Join("docs", strings.TrimPrefix(r.URL.Path, "/yoke-website/")))
			if err != nil {
				w.Header().Set("Content-Type", "text/html")
				if errors.Is(err, os.ErrNotExist) {
					w.WriteHeader(404)
					w.Write(knownPages.NotFound)
				} else {
					w.WriteHeader(500)
					w.Write(knownPages.InternalError)
				}
				return
			}

			w.Header().Set("Content-Type", mime.TypeByExtension(path.Ext(r.URL.Path)))
			w.Write(content)
		}
	}())

	handler := WithLogger(mux)

	return h2c.NewHandler(handler, new(http2.Server))
}

func WithLogger(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorder := CodeRecorder{ResponseWriter: w, code: 200}
		handler.ServeHTTP(&recorder, r)
		fmt.Printf("%s | %d %s %s\n", time.Now().Format("2006-01-02/15:04:05"), recorder.code, r.Method, r.URL)
	})
}

type CodeRecorder struct {
	http.ResponseWriter
	code int
}

func (recorder *CodeRecorder) WriteHeader(code int) {
	recorder.ResponseWriter.WriteHeader(code)
	recorder.code = code
}
