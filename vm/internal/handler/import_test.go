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
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"

	"github.com/docker/volumes-backup-extension/internal/backend"
)

// TestImportTarGzFileFromPreviousVersion tests the backwards compatibility of importing a tar.gz archive generated from version 1.0.0 of the Volumes Backup extension.
// See https://github.com/docker/volumes-backup-extension/pull/63.
func TestImportTarGzFileFromExtensionVersion1_0_0(t *testing.T) {
	volume := "d022054e9eff40145acba93f2787c3d91113319c3df7f9115e441f0ce2af167b"
	cli := setupDockerClient(t)

	defer func() {
		_ = cli.VolumeRemove(context.Background(), volume, true)
	}()

	fileName := "postgres_pgdata.tar.gz"

	// Setup
	e := echo.New()
	q := make(url.Values)
	pwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	absolutePath := filepath.Join(pwd, "testdata", "import", "1.0.0", fileName)
	q.Set("path", absolutePath)
	req := httptest.NewRequest(http.MethodGet, "/?"+q.Encode(), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/volumes/:volume/import")
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

	// Import volume
	err = h.ImportTarGzFile(c)

	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)
	sizes, err := backend.GetVolumesSize(c.Request().Context(), cli, volume)
	require.NoError(t, err)
	require.Equal(t, int64(50764000), sizes[volume].Bytes)
	require.Equal(t, "50.8 MB", sizes[volume].Human)
}

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
	h := New(c.Request().Context(), func() (*client.Client, error) { return setupDockerClient(t), nil })

	// Create volume
	_, err = cli.VolumeCreate(c.Request().Context(), volumetypes.VolumeCreateBody{
		Driver: "local",
		Name:   volume,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Import volume
	err = h.ImportTarGzFile(c)

	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)
	sizes, err := backend.GetVolumesSize(c.Request().Context(), cli, volume)
	require.NoError(t, err)
	require.Equal(t, int64(16000), sizes[volume].Bytes)
	require.Equal(t, "16.0 kB", sizes[volume].Human)
}

func TestImportTarGzFileShouldRemovePreviousVolumeData(t *testing.T) {
	volume := "2744b01d1dfca0353a9f717988518d03307a119fe34d6fe5948f8a984f7f8d1f"
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
	resp, err := cli.ContainerCreate(context.Background(), &container.Config{
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

	if err := cli.ContainerStart(context.Background(), resp.ID, types.ContainerStartOptions{}); err != nil {
		t.Fatal(err)
	}

	if err := cli.ContainerRemove(context.Background(), resp.ID, types.ContainerRemoveOptions{
		Force: true,
	}); err != nil {
		t.Fatal(err)
	}

	// Import volume
	err = h.ImportTarGzFile(c)

	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)
	sizes, err := backend.GetVolumesSize(c.Request().Context(), cli, volume)
	require.NoError(t, err)
	require.Equal(t, int64(16000), sizes[volume].Bytes)
	require.Equal(t, "16.0 kB", sizes[volume].Human)
}
