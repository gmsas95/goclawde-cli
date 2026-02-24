//go:build embed
// +build embed

package dashboard

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
)

//go:embed all:dist
var distFS embed.FS

// GetStaticFS returns the embedded filesystem for the dashboard
func GetStaticFS() (http.FileSystem, error) {
	staticFS, err := fs.Sub(distFS, "dist")
	if err != nil {
		return nil, err
	}
	return http.FS(staticFS), nil
}

// RegisterStatic registers the static file serving middleware
func RegisterStatic(app *fiber.App) error {
	staticFS, err := GetStaticFS()
	if err != nil {
		return err
	}

	// Serve static files
	app.Use("/", filesystem.New(filesystem.Config{
		Root:   staticFS,
		Browse: false,
		Index:  "index.html",
		MaxAge: 3600,
	}))

	return nil
}
