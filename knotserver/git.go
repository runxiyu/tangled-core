package knotserver

import (
	"compress/gzip"
	"io"
	"log"
	"net/http"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/icyphox/bild/knotserver/git/service"
)

func (d *Handle) InfoRefs(w http.ResponseWriter, r *http.Request) {
	did := chi.URLParam(r, "did")
	name := chi.URLParam(r, "name")
	repo := filepath.Join(d.c.Repo.ScanPath, did, name)

	w.Header().Set("content-type", "application/x-git-upload-pack-advertisement")
	w.WriteHeader(http.StatusOK)

	cmd := service.ServiceCommand{
		Dir:    repo,
		Stdout: w,
	}

	if err := cmd.InfoRefs(); err != nil {
		http.Error(w, err.Error(), 500)
		log.Printf("git: failed to execute git-upload-pack (info/refs) %s", err)
		return
	}
}

func (d *Handle) UploadPack(w http.ResponseWriter, r *http.Request) {
	did := chi.URLParam(r, "did")
	name := chi.URLParam(r, "name")
	repo := filepath.Join(d.c.Repo.ScanPath, did, name)

	w.Header().Set("content-type", "application/x-git-upload-pack-result")
	w.Header().Set("Connection", "Keep-Alive")
	w.Header().Set("Transfer-Encoding", "chunked")
	w.WriteHeader(http.StatusOK)

	cmd := service.ServiceCommand{
		Dir:    repo,
		Stdout: w,
	}

	var reader io.ReadCloser
	reader = r.Body

	if r.Header.Get("Content-Encoding") == "gzip" {
		reader, err := gzip.NewReader(r.Body)
		if err != nil {
			http.Error(w, err.Error(), 500)
			log.Printf("git: failed to create gzip reader: %s", err)
			return
		}
		defer reader.Close()
	}

	cmd.Stdin = reader
	if err := cmd.UploadPack(); err != nil {
		http.Error(w, err.Error(), 500)
		log.Printf("git: failed to execute git-upload-pack %s", err)
		return
	}
}
