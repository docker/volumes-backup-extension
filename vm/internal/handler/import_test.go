package handler

import (
	"context"
	volumetypes "github.com/docker/docker/api/types/volume"
	"github.com/felipecruz91/vackup-docker-extension/internal/backend"
	"github.com/labstack/echo"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func TestImportTarGzFile(t *testing.T) {
	volume := "9a66f7e879b539462d372feee03588aed95fe03236be950b0b1ed55ec7b995d1"
	cli := setupDockerClient(t)

	defer func() {
		_ = cli.VolumeRemove(context.Background(), volume, true)
	}()

	fileName := "nginx.tar.gz"

	// Setup
	e := echo.New()
	q := make(url.Values)
	pwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	absolutePath := filepath.Join(pwd, "testdata", "import", fileName)
	q.Set("path", absolutePath)
	req := httptest.NewRequest(http.MethodGet, "/?"+q.Encode(), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/volumes/:volume/import")
	c.SetParamNames("volume")
	c.SetParamValues(volume)
	h := New(c.Request().Context(), setupDockerClient(t))

	// Create volume
	_, err = cli.VolumeCreate(c.Request().Context(), volumetypes.VolumeCreateBody{
		Driver: "local",
		Name:   volume,
	})
	require.NoError(t, err)

	// Import volume
	err = h.ImportTarGzFile(c)

	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)
	sizes := backend.GetVolumesSize(c.Request().Context(), cli, volume)
	require.Regexp(t, ".*K", sizes[volume])
}
