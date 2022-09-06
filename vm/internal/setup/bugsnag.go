package setup

import (
	"net/http"
	"os"

	"github.com/bugsnag/bugsnag-go/v2"
	"github.com/felipecruz91/vackup-docker-extension/internal/log"
)

func ConfigureBugsnag() {
	bugsnagAPIKey := os.Getenv("BUGSNAG_API_KEY")
	if bugsnagAPIKey == "" {
		log.Warn(`Bugsnag configuration not added as environment variable "BUGSNAG_API_KEY" is empty.`)
		return
	}

	bugsnag.Configure(bugsnag.Configuration{
		APIKey:       bugsnagAPIKey,
		ReleaseStage: "production",
		// The import paths for the Go packages containing your source files
		ProjectPackages: []string{"main", "github.com/docker/volumes-backup-extension"},
		AppVersion:      os.Getenv("EXTENSION_IMAGE_TAG"),
	})

	log.Info("Bugsnag configuration added successfully.")
}

// ConfigureBugsnagHandler uses bugsnag.Handler(nil) to wrap the default http handlers
// so that Bugsnag is automatically notified about panics.
// See: https://docs.bugsnag.com/platforms/go/net-http/#basic-configuration
func ConfigureBugsnagHandler(server *http.Server) {
	bugsnagAPIKey := os.Getenv("BUGSNAG_API_KEY")
	if bugsnagAPIKey == "" {
		log.Warn(`Bugsnag handler to notify about panics not configured as environment variable "BUGSNAG_API_KEY" is empty.`)
		return
	}
	server.Handler = bugsnag.Handler(nil)
	log.Info("Bugsnag handler configured successfully.")
}
