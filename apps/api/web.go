package main

import (
	"bytes"
	"fmt"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/beambeambeam/claimflow/apps/api/internal/httperr"
)

const (
	webIndex  = "index.html"
	webAssets = "assets/"
	// Vite puts a content hash in every file name under assets/, so a cached copy never goes stale.
	immutableCache = "public, max-age=31536000, immutable"
)

// serveWebBuild serves the web build in files for GET and HEAD outside /api (SD-15). A path that names no
// file gets index.html, so the client router opens it. An unknown /api route, a missing asset, and any other
// method stay 404 problems.
func serveWebBuild(files fs.FS) gin.HandlerFunc {
	return func(c *gin.Context) {
		urlPath := c.Request.URL.Path
		isAPI := urlPath == "/api" || strings.HasPrefix(urlPath, "/api/")
		isRead := c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead

		if isAPI || !isRead {
			httperr.NotFound(c)

			return
		}

		// Clean drops every "..", so the name stays inside the build.
		name := strings.TrimPrefix(path.Clean("/"+urlPath), "/")
		isAsset := strings.HasPrefix(name, webAssets)

		data, err := fs.ReadFile(files, name)
		if err != nil {
			if isAsset {
				httperr.NotFound(c)

				return
			}

			name = webIndex

			data, err = fs.ReadFile(files, name)
			if err != nil {
				httperr.Report(c, fmt.Errorf("web build: %w", err))

				return
			}
		}

		switch {
		case name == webIndex:
			c.Header("Cache-Control", "no-cache")
		case isAsset:
			c.Header("Cache-Control", immutableCache)
		}

		http.ServeContent(c.Writer, c.Request, name, time.Time{}, bytes.NewReader(data))
	}
}
