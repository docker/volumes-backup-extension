package handler

import (
	"context"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	volumetypes "github.com/docker/docker/api/types/volume"
	"github.com/labstack/echo"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"testing"
)

func TestVolumeSize(t *testing.T) {
	var containerID string
	volume := "f1c149694ab1318377505d40c2431b4387a53d8d28fa814d4584e12b1ed63cfc"
	cli := setupDockerClient(t)

	defer func() {
		_ = cli.ContainerRemove(context.Background(), containerID, types.ContainerRemoveOptions{
			Force: true,
		})
		_ = cli.VolumeRemove(context.Background(), volume, true)
	}()

	// Setup
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/volumes/:volume/size")
	c.SetParamNames("volume")
	c.SetParamValues(volume)
	h := New(c.Request().Context(), setupDockerClient(t))

	// Create volume
	_, err := cli.VolumeCreate(c.Request().Context(), volumetypes.VolumeCreateBody{
		Driver: "local",
		Name:   volume,
	})
	if err != nil {
		t.Fatal(err)
	}

	reader, err := cli.ImagePull(c.Request().Context(), "docker.io/library/nginx:1.21", types.ImagePullOptions{
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
	resp, err := cli.ContainerCreate(c.Request().Context(), &container.Config{
		Image: "docker.io/library/nginx:1.21",
	}, &container.HostConfig{
		Binds: []string{
			volume + ":" + "/usr/share/nginx/html:ro",
		},
	}, nil, nil, "")
	if err != nil {
		t.Fatal(err)
	}

	containerID = resp.ID

	// Get volume size
	err = h.VolumeSize(c)
	require.NoError(t, err)

	size := rec.Body.String()
	t.Log(size)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Regexp(t, "\".*K\"\n", size)
}
