package dashboard

import (
	"net/http"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
)

// GetStaticFS returns the dashboard filesystem from disk
// Note: Files are expected to be at web/dashboard/dist relative to working directory
func GetStaticFS() (http.FileSystem, error) {
	paths := []string{
		"./web/dashboard/dist",
		"./web/dist",
		"/app/web/dashboard/dist",
		"/app/web",
	}

	for _, path := range paths {
		if stat, err := os.Stat(path); err == nil && stat.IsDir() {
			return http.Dir(path), nil
		}
	}

	return nil, os.ErrNotExist
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
