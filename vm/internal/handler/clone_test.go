package handler

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"runtime"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"

	"github.com/docker/volumes-backup-extension/internal/backend"
)

func TestCloneVolume(t *testing.T) {
	var containerID string
	volumeID := "e6b2874a1b4ced2344d53b75e93346f60e1c363fe3e4cd9c6cb5bd8b975b9a45"
	destVolume := volumeID + "-cloned"
	cli := setupDockerClient(t)

	defer func() {
		_ = cli.ContainerRemove(context.Background(), containerID, types.ContainerRemoveOptions{
			Force: true,
		})
		_ = cli.VolumeRemove(context.Background(), volumeID, true)
		_ = cli.VolumeRemove(context.Background(), destVolume, true)
	}()

	// Setup
	e := echo.New()
	q := make(url.Values)
	q.Set("destVolume", destVolume)
	req := httptest.NewRequest(http.MethodPost, "/?"+q.Encode(), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/volumes/:volume/clone")
	c.SetParamNames("volume")
	c.SetParamValues(volumeID)
	h := New(c.Request().Context(), func() (*client.Client, error) { return setupDockerClient(t), nil })

	// Create volume
	_, err := cli.VolumeCreate(c.Request().Context(), volume.CreateOptions{
		Driver: "local",
		Name:   volumeID,
		Labels: map[string]string{
			"com.docker.compose.project": "my-compose-project",
			"com.docker.compose.version": "2.10.2",
			"com.docker.compose.volume":  "foo-bar",
		},
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
			volumeID + ":" + "/usr/share/nginx/html:ro",
		},
	}, nil, nil, "")
	if err != nil {
		t.Fatal(err)
	}

	containerID = resp.ID

	// Clone volume
	err = h.CloneVolume(c)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, rec.Code)

	dockerClient, err := h.DockerClient()
	if err != nil {
		t.Fatal(err)
	}
	// Check volume has been cloned and contains the expected data
	clonedVolumeResp, err := dockerClient.VolumeList(context.Background(), volume.ListOptions{Filters: filters.NewArgs(filters.Arg("name", destVolume))})
	if err != nil {
		t.Fatal(err)
	}
	require.Len(t, clonedVolumeResp.Volumes, 1)
	sizes, err := backend.GetVolumesSize(context.Background(), dockerClient, destVolume)
	require.NoError(t, err)
	require.Equal(t, int64(16000), sizes[destVolume].Bytes)
	require.Equal(t, "16.0 kB", sizes[destVolume].Human)

	// Check volume labels
	volInspect, err := cli.VolumeInspect(context.Background(), destVolume)
	if err != nil {
		t.Fatal(err)
	}
	require.Len(t, volInspect.Labels, 3)
	require.Equal(t, "my-compose-project", volInspect.Labels["com.docker.compose.project"])
	require.Equal(t, "2.10.2", volInspect.Labels["com.docker.compose.version"])
	require.Equal(t, "foo-bar", volInspect.Labels["com.docker.compose.volume"])
}
