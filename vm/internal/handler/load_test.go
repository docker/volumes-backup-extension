package handler

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	volumetypes "github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/docker/volumes-backup-extension/internal/backend"
	"github.com/labstack/echo"
	"github.com/stretchr/testify/require"
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
	h := New(c.Request().Context(), func() (*client.Client, error) { return setupDockerClient(t), nil })

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
	t.Logf("Volume size after loading image into it: %+v", sizes[volume])
	require.Equal(t, int64(16000), sizes[volume].Bytes)
	require.Equal(t, "16.0 kB", sizes[volume].Human)
}

func TestLoadImageShouldRemovePreviousVolumeData(t *testing.T) {
	volume := "348cbc9bc7092dcdf4acdd3653ffd7711cbe1b529c6fd699ecddec5c3577613c"
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
	h := New(c.Request().Context(), func() (*client.Client, error) { return setupDockerClient(t), nil })

	// Create volume
	_, err = cli.VolumeCreate(c.Request().Context(), volumetypes.VolumeCreateBody{
		Driver: "local",
		Name:   volume,
	})
	if err != nil {
		t.Fatal(err)
	}

	reader, err := cli.ImagePull(c.Request().Context(), "docker.io/library/postgres:14-alpine", types.ImagePullOptions{
		Platform: "linux/" + runtime.GOARCH,
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = io.Copy(os.Stdout, reader)
	if err != nil {
		t.Fatal(err)
	}

	// Populate volume
	postgresResp, err := cli.ContainerCreate(context.Background(), &container.Config{
		Image: "docker.io/library/postgres:14-alpine",
		Env:   []string{"POSTGRES_PASSWORD=password"},
	}, &container.HostConfig{
		Binds: []string{
			volume + ":" + "/var/lib/postgresql/data",
		},
	}, nil, nil, "")
	if err != nil {
		t.Fatal(err)
	}

	if err := cli.ContainerStart(context.Background(), postgresResp.ID, types.ContainerStartOptions{}); err != nil {
		t.Fatal(err)
	}

	if err := cli.ContainerRemove(context.Background(), postgresResp.ID, types.ContainerRemoveOptions{
		Force: true,
	}); err != nil {
		t.Fatal(err)
	}

	// Load image into volume
	err = h.LoadImage(c)

	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)
	sizes := backend.GetVolumesSize(c.Request().Context(), cli, volume)
	t.Logf("Volume size after loading image into it: %+v", sizes[volume])
	require.Equal(t, int64(16000), sizes[volume].Bytes)
	require.Equal(t, "16.0 kB", sizes[volume].Human)
}
