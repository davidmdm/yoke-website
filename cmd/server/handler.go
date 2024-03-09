package main

import (
	"errors"
	"fmt"
	"io"
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
		if r.URL.Path == "/" {
			w.Write(knownPages.Home)
			return
		}
		w.WriteHeader(404)
		w.Write(knownPages.NotFound)
	})

	mux.HandleFunc("GET /content/", func() http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", mime.TypeByExtension(path.Ext(r.URL.Path)))

			content, err := content.ReadFile(r.URL.Path[1:])
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					w.WriteHeader(404)
					w.Write(knownPages.NotFound)
				} else {
					w.WriteHeader(500)
					w.Write(knownPages.InternalError)
				}
				return
			}

			w.Write(content)
		}
	}())

	handler := WithMethod(mux, "GET")
	handler = WithLogger(handler)

	return h2c.NewHandler(handler, new(http2.Server))
}

func WithMethod(handler http.Handler, method string) http.Handler {
	method = strings.ToUpper(method)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			w.WriteHeader(405)
			io.WriteString(w, "<html><body>methot not allowed</body></html>")
			return
		}
		handler.ServeHTTP(w, r)
	})
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

func Must(err error) {
	if err != nil {
		panic(err)
	}
}

func Must2[T any](value T, err error) T {
	Must(err)
	return value
}

type pageWriter struct {
	http.ResponseWriter
	code    int
	written bool
}

func (w *pageWriter) WriteHeader(code int) {
	w.code = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *pageWriter) Write(data []byte) (int, error) {
	if !w.written && (w.code == 404 || w.code == 500) {
		switch w.code {
		case 404:
			w.Header().Set("Content-Type", "text/html")
			w.ResponseWriter.Write(knownPages.NotFound)
		case 500:
			w.ResponseWriter.Write(knownPages.InternalError)
		}
		w.written = true
	}

	if w.written {
		return len(data), nil
	}

	return w.ResponseWriter.Write(data)
}
