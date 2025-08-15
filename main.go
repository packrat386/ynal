package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"mime"
	"net/http"
	"os"
	"path"
	"slices"
	"strconv"
	"strings"
)

//go:embed public/*
var publicFS embed.FS

//go:embed licenses/*
var licensesFS embed.FS

//go:embed templates/*
var templatesFS embed.FS

func main() {
	h, err := appHandler()
	if err != nil {
		panic(err)
	}

	srv := http.Server{
		Addr:    addr(),
		Handler: withLogging(h),
	}

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		panic(err)
	}
}

func addr() string {
	if val := os.Getenv("YNAL_ADDR"); val != "" {
		return val
	} else {
		return "localhost:8080"
	}
}

func appHandler() (http.Handler, error) {
	tmpl, err := template.ParseFS(templatesFS, "templates/*.tmpl")
	if err != nil {
		return nil, fmt.Errorf("could not parse templates: %w", err)
	}

	public, err := fs.Sub(publicFS, "public")
	if err != nil {
		return nil, fmt.Errorf("could not subsystem public assets: %w", err)
	}

	licenses, err := fs.Glob(licensesFS, "licenses/*.txt")
	if err != nil {
		return nil, fmt.Errorf("could not glob licenses: %w", err)
	}

	mux := http.NewServeMux()

	supported := []LicenseData{}

	for _, lpath := range licenses {
		h, data, err := handlerFor(lpath, licensesFS, tmpl)
		if err != nil {
			return nil, fmt.Errorf("could not init handler: %w", err)
		}

		mux.Handle("GET "+data.URL, h)
		supported = append(supported, data)
	}

	mux.Handle("/", newPublicHandler(public, tmpl, supported))

	return mux, nil
}

func pathToURL(lpath string) string {
	return "/" + strings.ToLower(pathToTitle(lpath))
}

func pathToTitle(lpath string) string {
	return strings.TrimSuffix(path.Base(lpath), path.Ext(lpath))
}

func handlerFor(lpath string, licenses embed.FS, tmpl *template.Template) (http.Handler, LicenseData, error) {
	plainData, err := fs.ReadFile(licenses, lpath)
	if err != nil {
		return nil, LicenseData{}, fmt.Errorf("could not read license: %w", err)
	}

	l := LicenseData{
		Title: pathToTitle(lpath),
		Text:  string(plainData),
		URL:   pathToURL(lpath),
	}

	htmlData, err := toHTML(l, tmpl)
	if err != nil {
		return nil, LicenseData{}, fmt.Errorf("could not render HTML: %w", err)
	}

	jsonData, err := toJSON(l)
	if err != nil {
		return nil, LicenseData{}, fmt.Errorf("could not render JSON: %w", err)
	}

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mediatype := mostAcceptable(r.Header.Get("Accept"))

		switch mediatype {
		case "text/plain":
			w.Header().Set("Content-Type", "text/plain")
			w.Write(plainData)
		case "text/html":
			w.Header().Set("Content-Type", "text/html")
			w.Write(htmlData)
		case "application/json":
			w.Header().Set("Content-Type", "application/json")
			w.Write(jsonData)
		default:
			http.Error(w, fmt.Sprintf("unrecognized media type: %s", mediatype), http.StatusNotAcceptable)
		}
	})

	return h, l, nil
}

type LicenseData struct {
	Title string `json:"title"`
	Text  string `json:"content"`
	URL   string `json:"url"`
}

func toHTML(l LicenseData, tmpl *template.Template) ([]byte, error) {
	buf := new(bytes.Buffer)

	err := tmpl.ExecuteTemplate(buf, "license.html.tmpl", l)
	if err != nil {
		return nil, fmt.Errorf("could not render html template: %w", err)
	}

	return buf.Bytes(), nil
}

func toJSON(l LicenseData) ([]byte, error) {
	b, err := json.Marshal(l)
	if err != nil {
		return nil, fmt.Errorf("could not marshal JSON: %w", err)
	}

	return b, nil
}

type AcceptType struct {
	MediaType      string
	RelativeWeight float64
}

func mostAcceptable(accept string) string {
	options := strings.Split(accept, ",")
	acceptable := []AcceptType{}

	for _, v := range options {
		mediatype, params, err := mime.ParseMediaType(v)
		if err != nil {
			continue
		}

		weight := float64(1.0)
		if val, err := strconv.ParseFloat(params["q"], 64); err == nil {
			weight = val
		}

		acceptable = append(acceptable, AcceptType{MediaType: mediatype, RelativeWeight: weight})
	}

	// kieras forgive me...
	slices.SortStableFunc(acceptable, func(a AcceptType, b AcceptType) int {
		if a.RelativeWeight > b.RelativeWeight {
			return -1
		} else if a.RelativeWeight < b.RelativeWeight {
			return 1
		} else {
			return 0
		}
	})

	for _, a := range acceptable {
		switch a.MediaType {
		case "text/plain", "*/*":
			return "text/plain"
		case "text/html", "application/xhtml+xml":
			return "text/html"
		case "application/json":
			return "application/json"
		}
	}

	// if nothing they sent matches, default to text/plain
	return "text/plain"
}

func newPublicHandler(public fs.FS, tmpl *template.Template, supported []LicenseData) http.Handler {
	buf := new(bytes.Buffer)

	tmpl.ExecuteTemplate(buf, "index.html.tmpl", supported)

	fileserver := http.FileServer(http.FS(public))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.Write(buf.Bytes())
		} else {
			fileserver.ServeHTTP(w, r)
		}
	})
}

type loggingResponseWriter struct {
	http.ResponseWriter
	code int
}

func (l *loggingResponseWriter) WriteHeader(code int) {
	l.code = code
	l.ResponseWriter.WriteHeader(code)
}

func withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lrw := &loggingResponseWriter{w, 200}

		next.ServeHTTP(lrw, r)

		log.Printf("%s [%d] %s", r.Method, lrw.code, r.URL.String())
	})
}
