package handler

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	volumetypes "github.com/docker/docker/api/types/volume"
	"github.com/docker/go-connections/nat"
	"github.com/felipecruz91/vackup-docker-extension/internal/backend"
	"github.com/labstack/echo"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPullVolume(t *testing.T) {
	var containerID string
	var registryContainerID string
	volume := "797a9e23f9da19b7c59e816c425bb491231914619a5d815435de9d8b28063bc8"
	registry := "localhost:5000" // or use docker.io to pull volume from DockerHub
	imageID := registry + "/felipecruz/" + "vackup-pull-test-img"
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

	// Provision a registry with an image (which represents a volume) ready to pull:
	// Run a local registry
	resp2, err := cli.ContainerCreate(context.Background(), &container.Config{
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

	if err := cli.ContainerStart(context.Background(), registryContainerID, types.ContainerStartOptions{}); err != nil {
		t.Fatal(err)
	}

	// Load .tar.gz filesystem into a local image
	pwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	fileName := "vackup-pull-test-img.tar.gz"
	absolutePath := filepath.Join(pwd, "testdata", "pull", fileName)
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
	requestJSON := fmt.Sprintf(`{"reference": "%s"}`, imageID)
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(requestJSON))
	req.Header.Add("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/volumes/:volume/pull")
	c.SetParamNames("volume")
	c.SetParamValues(volume)
	h := New(c.Request().Context(), setupDockerClient(t))

	//Push volume to registry.
	// The registry container state may be "running" but not ready to receive requests when reaching this line.
	err = retry(10, 1*time.Second, func() error {
		pushResp, err := cli.ImagePush(context.Background(), imageID, dockertypes.ImagePushOptions{
			RegistryAuth: "Cg==", // from running: echo "" | base64,
		})
		if err != nil {
			return err
		}
		defer pushResp.Close()

		response, err := ioutil.ReadAll(pushResp)
		if err != nil {
			return err
		}

		t.Log(string(response))

		if strings.Contains(string(response), "error") {
			return err
		}

		return nil
	})

	if err != nil {
		t.Fatal(err)
	}
	//for {
	//	containers, err := cli.ContainerList(c.Request().Context(), types.ContainerListOptions{
	//		Quiet:   false,
	//		All:     true,
	//		Limit:   1,
	//		Filters: filters.NewArgs(filters.Arg("id", registryContainerID)),
	//	})
	//	if err != nil {
	//		t.Fatal(err)
	//	}
	//
	//	if containers[0].State == "running" {
	//		break
	//	}
	//}

	_, err = cli.ImageRemove(context.Background(), imageID, types.ImageRemoveOptions{
		Force: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Create empty volume where the content of the image pulled will be saved into
	_, err = cli.VolumeCreate(c.Request().Context(), volumetypes.VolumeCreateBody{
		Driver: "local",
		Name:   volume,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Pull volume from registry
	err = h.PullVolume(c)

	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, rec.Code)

	// Check the content of the volume
	m := backend.GetVolumesSize(c.Request().Context(), h.DockerClient, volume)
	require.Equal(t, int64(16000), m[volume].Bytes)
	require.Equal(t, "16.0 kB", m[volume].Human)

}

func retry(attempts int, sleep time.Duration, f func() error) (err error) {
	for i := 0; i < attempts; i++ {
		if i > 0 {
			log.Println("retrying after error:", err)
			time.Sleep(sleep)
			sleep *= 2
		}
		err = f()
		if err == nil {
			return nil
		}
	}
	return fmt.Errorf("after %d attempts, last error: %s", attempts, err)
}

// TODO:
//func TestPullVolumeUsingCorrectAuth(t *testing.T) {
//}
//

// TODO:
//func TestPullVolumeUsingWrongAuthShouldFail(t *testing.T) {
//}
