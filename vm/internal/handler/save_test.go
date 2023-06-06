package handler

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

func TestSaveVolume(t *testing.T) {
	var containerID string
	volumeID := "cde5adac7d16ae45c6d2bf8f2496d3da3b994227bfa4d3ea392a03c2ad33cce6"
	imageID := "vackup-cde5adac7d16ae45c6d2bf8f2496d3da3b994227bfa4d3ea392a03c2ad33cce6:latest"
	cli := setupDockerClient(t)

	defer func() {
		_ = cli.ContainerRemove(context.Background(), containerID, types.ContainerRemoveOptions{
			Force: true,
		})
		_ = cli.VolumeRemove(context.Background(), volumeID, true)

		t.Logf("removing image %s", imageID)
		if _, err := cli.ImageRemove(context.Background(), imageID, types.ImageRemoveOptions{
			Force: true,
		}); err != nil {
			t.Log(err)
		}
	}()

	// Setup
	e := echo.New()
	q := make(url.Values)
	q.Set("image", imageID)
	req := httptest.NewRequest(http.MethodGet, "/?"+q.Encode(), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/volumes/:volume/save")
	c.SetParamNames("volume")
	c.SetParamValues(volumeID)
	h := New(c.Request().Context(), func() (*client.Client, error) { return cli, nil })

	// Create volume
	_, err := cli.VolumeCreate(c.Request().Context(), volume.CreateOptions{
		Driver: "local",
		Name:   volumeID,
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

	// Save volume
	err = h.SaveVolume(c)

	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, rec.Code)

	// Check the image exists
	summary, err := cli.ImageList(context.Background(), types.ImageListOptions{
		All:     false,
		Filters: filters.NewArgs(filters.Arg("reference", imageID)),
	})
	if err != nil {
		t.Fatal(err)
	}

	require.Len(t, summary, 1)
	require.Equal(t, imageID, summary[0].RepoTags[0])
	t.Logf("Image size after saving volume into it: %d", summary[0].Size)
	require.Regexp(t, `\d{7}`, strconv.FormatInt(summary[0].Size, 10), "the image size should be between 1 and 10 MB")
}
