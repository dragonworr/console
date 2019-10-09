package main

import (
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"go.uber.org/zap"
)

// getIndexFile loads and prepares the template index.html.
// Before returning, this method injects the needed server variables (like
// for example __COMMIT_SHA__) so the file is ready to be sent to a connecting browser.
// If the file is missing, we'll log a fatal error and exit with an error code.
func (api *API) getIndexFile(filePath string) []byte {
	indexPath := filePath + "/index.html"
	index, err := ioutil.ReadFile(indexPath)
	if err != nil {
		api.logger.Fatal("cannot read index.html", zap.String("path", indexPath), zap.Error(err))
		os.Exit(1)
	}

	index = []byte(strings.Replace(string(index), "__COMMIT_SHA__", commitSha, 1))

	if len(version) > 0 {
		index = []byte(strings.Replace(string(index), "__VERSION__", version, 1))
	}

	return index
}

// handleGetStaticFile tries to open the requested file. If this file does not exist it will return the
// SPA (index.html) instead.
func (api *API) handleGetStaticFile(index []byte, rootPath string) http.HandlerFunc {
	root := http.Dir(rootPath)
	fs := http.StripPrefix("/", http.FileServer(root))

	return func(w http.ResponseWriter, r *http.Request) {
		f, err := root.Open(r.RequestURI)
		if os.IsNotExist(err) {
			api.logger.Debug("requested file not found", zap.String("file", r.RequestURI))
			// everything else goes to index as well
			w.Write(index)
			return
		}
		defer f.Close()

		fs.ServeHTTP(w, r)
	}
}

// handleGetIndex returns the SPA (index.html)
func (api *API) handleGetIndex(index []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write(index)
		if err != nil {
			api.logger.Error("failed to write index file to response writer", zap.Error(err))
		}
	}
}
