package setup

import (
	"net/http"
	"os"
	"os/exec"
	"runtime"

	"github.com/labstack/echo/v4"

	"github.com/docker/volumes-backup-extension/internal/log"

	"github.com/bugsnag/bugsnag-go/v2"
)

func ConfigureBugsnag() {
	bugsnagAPIKey := os.Getenv("BUGSNAG_API_KEY")
	if bugsnagAPIKey == "" {
		log.Warn(`Bugsnag configuration not added as environment variable "BUGSNAG_API_KEY" is empty.`)
		return
	}

	bugsnag.Configure(bugsnag.Configuration{
		APIKey:       bugsnagAPIKey,
		ReleaseStage: os.Getenv("BUGSNAG_RELEASE_STAGE"),
		// The import paths for the Go packages containing your source files
		ProjectPackages: []string{"main", "github.com/docker/volumes-backup-extension"},
		AppVersion:      os.Getenv("BUGSNAG_APP_VERSION"),
	})

	bugsnag.OnBeforeNotify(func(event *bugsnag.Event, config *bugsnag.Configuration) error {
		event.MetaData.Add("OS", "Architecture", runtime.GOARCH)
		event.MetaData.Add("OS", "Docker Desktop Version", getDockerDesktopVersion())
		return nil
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

func ConfigureBugsnagHTTPErrorHandler(err error, c echo.Context) {
	if os.Getenv("BUGSNAG_API_KEY") == "" {
		return
	}

	he, ok := err.(*echo.HTTPError)
	if ok && he.Code != http.StatusInternalServerError {
		return
	}

	log.Error(err)
	_ = bugsnag.Notify(err, c.Request().Context())
}

func getDockerDesktopVersion() string {
	cmd := exec.Command("docker", "version", "--format", "{{ json .Server.Platform.Name }}") // e.g. "Docker Desktop 4.12.0 (85790)"
	stdout, err := cmd.Output()
	if err != nil {
		log.Error(err)
		return ""
	}

	return string(stdout)
}
