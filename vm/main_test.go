package main

import (
	"context"
	"encoding/json"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	volumetypes "github.com/docker/docker/api/types/volume"
	"github.com/labstack/echo"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func Test_volumes(t *testing.T) {
	volume := "7115890bd1cdf80f4cc0b8aaa9f5300281e80b4bf68170a6eb20e174774f0089"

	defer func() {
		_ = h.cli.VolumeRemove(context.Background(), volume, true)
	}()

	// Setup
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/volumes")

	// Create volume
	_, err := h.cli.VolumeCreate(c.Request().Context(), volumetypes.VolumeCreateBody{
		Driver: "local",
		Name:   volume,
	})
	require.NoError(t, err)

	// List volumes
	err = h.volumes(c)
	require.NoError(t, err)

	t.Log(rec.Body.String())
	m := map[string]VolumeData{}
	err = json.Unmarshal(rec.Body.Bytes(), &m)
	require.NoError(t, err)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, m, volume)
	require.Equal(t, "local", m[volume].Driver)
	require.Equal(t, "0B", m[volume].Size)
	require.Len(t, m[volume].Containers, 0)
}

func Test_volumeSize(t *testing.T) {
	var containerID string
	volume := "f1c149694ab1318377505d40c2431b4387a53d8d28fa814d4584e12b1ed63cfc"

	defer func() {
		_ = h.cli.ContainerRemove(context.Background(), containerID, types.ContainerRemoveOptions{
			Force: true,
		})
		_ = h.cli.VolumeRemove(context.Background(), volume, true)
	}()

	// Setup
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/volumes/:volume/size")
	c.SetParamNames("volume")
	c.SetParamValues(volume)

	// Create volume
	_, err := h.cli.VolumeCreate(c.Request().Context(), volumetypes.VolumeCreateBody{
		Driver: "local",
		Name:   volume,
	})
	require.NoError(t, err)

	reader, err := h.cli.ImagePull(c.Request().Context(), "docker.io/library/nginx:1.21", types.ImagePullOptions{})
	_, err = io.Copy(os.Stdout, reader)
	require.NoError(t, err)

	// Populate volume
	resp, err := h.cli.ContainerCreate(c.Request().Context(), &container.Config{
		Image: "docker.io/library/nginx:1.21",
	}, &container.HostConfig{
		Binds: []string{
			volume + ":" + "/usr/share/nginx/html:ro",
		},
	}, nil, nil, "")
	require.NoError(t, err)

	containerID = resp.ID

	// Get volume size
	err = h.volumeSize(c)
	require.NoError(t, err)

	size := rec.Body.String()
	t.Log(size)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Regexp(t, "\".*K\"\n", size)
}
