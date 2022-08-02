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

func TestLoadImage(t *testing.T) {
	volume := "ec654aa5062241db227476aa877efc67d22fa4d3f8ed759c7a9738afce417c71"
	cli := setupDockerClient(t)
	imageID := "vackup-load-test-img:latest"

	defer func() {
		_ = cli.VolumeRemove(context.Background(), volume, true)
	}()

	// Load image.tar.gz filesystem into a local image
	pwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	fileName := "vackup-load-test-img.tar.gz"
	absolutePath := filepath.Join(pwd, "testdata", "load", fileName)
	r, err := os.Open(absolutePath)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	resp, err := cli.ImageLoad(context.Background(), r, true)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	// Setup
	e := echo.New()
	q := make(url.Values)
	q.Set("image", imageID)
	req := httptest.NewRequest(http.MethodGet, "/?"+q.Encode(), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/volumes/:volume/load")
	c.SetParamNames("volume")
	c.SetParamValues(volume)
	h := New(c.Request().Context(), setupDockerClient(t))

	// Create volume
	_, err = cli.VolumeCreate(c.Request().Context(), volumetypes.VolumeCreateBody{
		Driver: "local",
		Name:   volume,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Load image into volume
	err = h.LoadImage(c)

	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)
	sizes := backend.GetVolumesSize(c.Request().Context(), cli, volume)
	t.Logf("Volume size after loading image into it: %s", sizes[volume])
	require.Regexp(t, ".*K", sizes[volume])
}
