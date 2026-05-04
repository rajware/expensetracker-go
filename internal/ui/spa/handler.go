package spa

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
)

//go:embed static
var staticFiles embed.FS

func NewHandler() http.Handler {
	sub, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Fatalln(err)
	}
	return http.FileServer(http.FS(sub))
}
