package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func assertEqualToFile(t *testing.T, r io.Reader, f string) {
	fdata, err := os.ReadFile(filepath.Join("testdata", f))
	if err != nil {
		t.Fatalf("error reading testdata: %s", err.Error())
	}

	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("error reading response data: %s", err.Error())
	}

	if string(fdata) != string(got) {
		t.Fatalf("expected:\n---\n%s\n---\ngot:\n---\n%s\n---\n", string(fdata), string(got))
	}
}

func mustAppHandler(t *testing.T) http.Handler {
	h, err := appHandler()
	if err != nil {
		t.Fatalf("could not initialize app handler: %s", err)
	}

	return h
}

func TestGetLicense(t *testing.T) {
	h := mustAppHandler(t)

	tt := []struct {
		name     string
		accept   string
		expected string
	}{
		{
			name:     "explicit plain text",
			accept:   "text/plain",
			expected: "EXPECTED_TXT",
		},
		{
			name:     "explicit html",
			accept:   "text/html",
			expected: "EXPECTED_HTML",
		},
		{
			name:     "explicit json",
			accept:   "application/json",
			expected: "EXPECTED_JSON",
		},
		{
			name:     "accept any",
			accept:   "*/*",
			expected: "EXPECTED_TXT",
		},
		{
			name:     "unrecognized",
			accept:   "lol/wut",
			expected: "EXPECTED_TXT",
		},
		{
			name:     "accept heirarchy",
			accept:   "text/css, text/plain; q=0.8, text/html; q=0.9",
			expected: "EXPECTED_HTML",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/mit", nil)
			r.Header.Set("Accept", tc.accept)

			w := httptest.NewRecorder()

			h.ServeHTTP(w, r)

			assertEqualToFile(t, w.Result().Body, tc.expected)
		})
	}
}
