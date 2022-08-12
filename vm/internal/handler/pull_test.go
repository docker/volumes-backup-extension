package handler

import (
	"context"
	"crypto/tls"
	"encoding/base64"
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

	// Push volume to registry.
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

	// The sleep is to ensure the image is present in the registry after the `ImagePush` operation.
	time.Sleep(3 * time.Second)

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

func TestPullVolumeUsingCorrectAuth(t *testing.T) {
	var containerID string
	var registryContainerID string
	volume := "b3128131acca4d70263d477345b528ad3aba3ae66f2d94dddb02817da427020a"
	registry := "localhost:5000" // or use docker.io to push it to DockerHub
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

	auth := `{"username": "testuser", "password": "testpassword"}`
	encodedAuth := base64.StdEncoding.EncodeToString([]byte(auth))

	// Setup
	e := echo.New()
	requestJSON := fmt.Sprintf(`{"reference": "%s"}`, imageID)
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(requestJSON))
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-Registry-Auth", encodedAuth)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/volumes/:volume/pull")
	c.SetParamNames("volume")
	c.SetParamValues(volume)
	h := New(c.Request().Context(), setupDockerClient(t))

	// Run a local registry with auth
	pwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	resp2, err := cli.ContainerCreate(context.Background(), &container.Config{
		Image: "docker.io/library/registry:2",
		ExposedPorts: map[nat.Port]struct{}{
			"5000/tcp": {},
		},
		Env: []string{
			"REGISTRY_AUTH=htpasswd",
			"REGISTRY_AUTH_HTPASSWD_REALM=Registry Realm",
			"REGISTRY_AUTH_HTPASSWD_PATH=/auth/htpasswd",
			"REGISTRY_HTTP_TLS_CERTIFICATE=/certs/domain.crt",
			"REGISTRY_HTTP_TLS_KEY=/certs/domain.key",
		},
	}, &container.HostConfig{
		Binds: []string{
			filepath.Join(pwd, "testdata", "push", "auth") + ":/auth",
			filepath.Join(pwd, "testdata", "push", "certs") + ":/certs",
		},
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

	// Push volume to registry.
	// The registry container state may be "running" but not ready to receive requests when reaching this line.
	err = retry(10, 1*time.Second, func() error {
		pushResp, err := cli.ImagePush(context.Background(), imageID, dockertypes.ImagePushOptions{
			RegistryAuth: encodedAuth,
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

	// The sleep is to ensure the image is present in the registry after the `ImagePush` operation.
	time.Sleep(3 * time.Second)

	_, err = cli.ImageRemove(context.Background(), imageID, types.ImageRemoveOptions{
		Force: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Create empty volume where the content of the image pulled will be saved into
	_, err = cli.VolumeCreate(context.Background(), volumetypes.VolumeCreateBody{
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

func TestPullVolumeUsingWrongAuthShouldFail(t *testing.T) {
	var containerID string
	var registryContainerID string
	volume := "98fbf55c3ab45ea5bfc8ea8edf7f374b325e68eab947828aae5d6447df19ee3d"
	registry := "localhost:5000" // or use docker.io to push it to DockerHub
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

	auth := `{"username": "testuser", "password": "wrongpassword"}`
	encodedAuth := base64.StdEncoding.EncodeToString([]byte(auth))

	// Setup
	e := echo.New()
	requestJSON := fmt.Sprintf(`{"reference": "%s"}`, imageID)
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(requestJSON))
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-Registry-Auth", encodedAuth)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/volumes/:volume/pull")
	c.SetParamNames("volume")
	c.SetParamValues(volume)
	h := New(c.Request().Context(), setupDockerClient(t))

	// Run a local registry with auth
	pwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	resp2, err := cli.ContainerCreate(context.Background(), &container.Config{
		Image: "docker.io/library/registry:2",
		ExposedPorts: map[nat.Port]struct{}{
			"5000/tcp": {},
		},
		Env: []string{
			"REGISTRY_AUTH=htpasswd",
			"REGISTRY_AUTH_HTPASSWD_REALM=Registry Realm",
			"REGISTRY_AUTH_HTPASSWD_PATH=/auth/htpasswd",
			"REGISTRY_HTTP_TLS_CERTIFICATE=/certs/domain.crt",
			"REGISTRY_HTTP_TLS_KEY=/certs/domain.key",
		},
	}, &container.HostConfig{
		Binds: []string{
			filepath.Join(pwd, "testdata", "push", "auth") + ":/auth",
			filepath.Join(pwd, "testdata", "push", "certs") + ":/certs",
		},
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

	// The registry container state may be "running" but not ready to receive requests when reaching this line.
	err = retry(10, 1*time.Second, func() error {
		req, err := http.NewRequest("GET", "https://localhost:5000/v2/_catalog", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Add("Authorization", "Basic "+basicAuth("testuser", "testpassword"))

		httpClient := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		}
		catalogResp, err := httpClient.Do(req)
		if err != nil {
			return err
		}
		defer catalogResp.Body.Close()
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// Create empty volume where the content of the image pulled will be saved into
	_, err = cli.VolumeCreate(context.Background(), volumetypes.VolumeCreateBody{
		Driver: "local",
		Name:   volume,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Pull volume from registry
	err = h.PullVolume(c)

	require.NoError(t, err)
	require.Equal(t, http.StatusUnauthorized, rec.Code)

	// Check the content of the volume
	m := backend.GetVolumesSize(c.Request().Context(), h.DockerClient, volume)
	require.Equal(t, int64(0), m[volume].Bytes)
	require.Equal(t, "0 B", m[volume].Human)
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
