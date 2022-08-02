package handler

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	volumetypes "github.com/docker/docker/api/types/volume"
	"github.com/docker/go-connections/nat"
	"github.com/labstack/echo"
	"github.com/stretchr/testify/require"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"strings"
	"testing"
)

func TestPushVolume(t *testing.T) {
	var containerID string
	var registryContainerID string
	volume := "998c9e00ea6ed7f7d27beeb2d876b18a02686172faa8897c50720c1365c82d8f"
	registry := "localhost:5000" // or use docker.io to push it to DockerHub
	imageID := registry + "/felipecruz/" + "test-push-volume-as-image"
	cli := setupDockerClient(t)

	defer func() {
		_ = cli.ContainerRemove(context.Background(), registryContainerID, types.ContainerRemoveOptions{
			Force: true,
		})
		_ = cli.ContainerRemove(context.Background(), containerID, types.ContainerRemoveOptions{
			Force: true,
		})
		_ = cli.VolumeRemove(context.Background(), volume, true)

		t.Logf("removing image %s", imageID)
		if _, err := cli.ImageRemove(context.Background(), imageID, types.ImageRemoveOptions{
			Force: true,
		}); err != nil {
			t.Log(err)
		}
	}()

	// Setup
	e := echo.New()
	requestJSON := fmt.Sprintf(`{"reference": "%s"}`, imageID)
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(requestJSON))
	req.Header.Add("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/volumes/:volume/push")
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

	// Run a local registry
	resp2, err := cli.ContainerCreate(c.Request().Context(), &container.Config{
		Image: "docker.io/library/registry:2",
		ExposedPorts: map[nat.Port]struct{}{
			"5000/tcp": {},
		},
	}, &container.HostConfig{
		PortBindings: map[nat.Port][]nat.PortBinding{
			"5000/tcp": {
				{
					HostPort: "5000",
				},
			},
		},
	}, nil, nil, "")
	if err != nil {
		t.Fatal(err)
	}

	registryContainerID = resp2.ID

	if err := h.DockerClient.ContainerStart(c.Request().Context(), registryContainerID, types.ContainerStartOptions{}); err != nil {
		t.Fatal(err)
	}

	// Push volume
	err = h.PushVolume(c)

	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, rec.Code)

	// Check the image exists in the registry
	catalogResp, err := http.Get("http://localhost:5000/v2/_catalog")
	if err != nil {
		t.Fatal(err)
	}
	defer catalogResp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}

	if catalogResp.StatusCode != http.StatusOK {
		t.Fatalf("status code: %d", catalogResp.StatusCode)
	}

	body, err := ioutil.ReadAll(catalogResp.Body)
	if err != nil {
		t.Fatal(err)
	}

	require.Equal(t, `{"repositories":["felipecruz/test-push-volume-as-image"]}
`, string(body))
}
