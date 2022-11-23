package handler

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	volumetypes "github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
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
	requestJSON := fmt.Sprintf(`{"reference": "%s", "base64EncodedAuth": ""}`, imageID)
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(requestJSON))
	req.Header.Add("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/volumes/:volume/push")
	c.SetParamNames("volume")
	c.SetParamValues(volume)
	h := New(c.Request().Context(), func() (*client.Client, error) { return setupDockerClient(t), nil })

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
	reader, err = cli.ImagePull(context.Background(), "docker.io/library/registry:2", types.ImagePullOptions{
		Platform: "linux/" + runtime.GOARCH,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = io.Copy(os.Stdout, reader)
	if err != nil {
		t.Fatal(err)
	}

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

	if err := cli.ContainerStart(c.Request().Context(), registryContainerID, types.ContainerStartOptions{}); err != nil {
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

func TestPushVolumeUsingCorrectAuth(t *testing.T) {
	var containerID string
	var registryContainerID string
	volume := "26c3626656572089590620f155e0b097309ab5c53e5ce6fba94cf8ed94e0dfb7"
	registry := "localhost:5000" // or use docker.io to push it to DockerHub
	imageID := registry + "/felipecruz/" + "test-auth-push-volume-as-image"
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
	requestJSON := fmt.Sprintf(`{"reference": "%s", "base64EncodedAuth": "%s"}`, imageID, encodedAuth)
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(requestJSON))
	req.Header.Add("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/volumes/:volume/push")
	c.SetParamNames("volume")
	c.SetParamValues(volume)
	h := New(c.Request().Context(), func() (*client.Client, error) { return setupDockerClient(t), nil })

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
	pwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	reader, err = cli.ImagePull(context.Background(), "docker.io/library/registry:2", types.ImagePullOptions{
		Platform: "linux/" + runtime.GOARCH,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = io.Copy(os.Stdout, reader)
	if err != nil {
		t.Fatal(err)
	}

	resp2, err := cli.ContainerCreate(c.Request().Context(), &container.Config{
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

	if err := cli.ContainerStart(c.Request().Context(), registryContainerID, types.ContainerStartOptions{}); err != nil {
		t.Fatal(err)
	}

	// Push volume
	err = h.PushVolume(c)

	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, rec.Code)

	// Check the image exists in the registry
	req, err = http.NewRequest("GET", "https://localhost:5000/v2/_catalog", nil)
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
		t.Fatal(err)
	}
	defer catalogResp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}

	body, err := ioutil.ReadAll(catalogResp.Body)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(string(body))

	require.Equal(t, `{"repositories":["felipecruz/test-auth-push-volume-as-image"]}
`, string(body))
}

func TestPushVolumeUsingWrongAuthShouldFail(t *testing.T) {
	var containerID string
	var registryContainerID string
	volume := "a5783caca4a98259c6d5a493e240f227f1aa93e72afd0fecaeb3a5575b8505d2"
	registry := "localhost:5000" // or use docker.io to push it to DockerHub
	imageID := registry + "/felipecruz/" + "test-auth-push-volume-as-image"
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
	requestJSON := fmt.Sprintf(`{"reference": "%s", "base64EncodedAuth": "%s"}`, imageID, encodedAuth)
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(requestJSON))
	req.Header.Add("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/volumes/:volume/push")
	c.SetParamNames("volume")
	c.SetParamValues(volume)
	h := New(c.Request().Context(), func() (*client.Client, error) { return setupDockerClient(t), nil })

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
	pwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	reader, err = cli.ImagePull(c.Request().Context(), "docker.io/library/registry:2", types.ImagePullOptions{
		Platform: "linux/" + runtime.GOARCH,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = io.Copy(os.Stdout, reader)
	if err != nil {
		t.Fatal(err)
	}

	resp2, err := cli.ContainerCreate(c.Request().Context(), &container.Config{
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

	if err := cli.ContainerStart(c.Request().Context(), registryContainerID, types.ContainerStartOptions{}); err != nil {
		t.Fatal(err)
	}

	// Push volume
	err = h.PushVolume(c)

	require.NoError(t, err)
	require.Equal(t, http.StatusUnauthorized, rec.Code)

	// Check the image does NOT exist in the registry due to the failed auth
	req, err = http.NewRequest("GET", "https://localhost:5000/v2/_catalog", nil)
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
		t.Fatal(err)
	}
	defer catalogResp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}

	body, err := ioutil.ReadAll(catalogResp.Body)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(string(body))

	require.Equal(t, `{"repositories":[]}
`, string(body))
}

func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}
