package handler

import (
	"context"
	"encoding/json"
	volumetypes "github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/labstack/echo"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVolumes(t *testing.T) {
	volume := "7115890bd1cdf80f4cc0b8aaa9f5300281e80b4bf68170a6eb20e174774f0089"
	cli := setupDockerClient(t)

	defer func() {
		_ = cli.VolumeRemove(context.Background(), volume, true)
	}()

	// Setup
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/volumes")
	h := New(c.Request().Context(), setupDockerClient(t))

	// Create volume
	_, err := cli.VolumeCreate(c.Request().Context(), volumetypes.VolumeCreateBody{
		Driver: "local",
		Name:   volume,
	})
	require.NoError(t, err)

	// List volumes
	err = h.Volumes(c)
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

func setupDockerClient(t *testing.T) *client.Client {
	t.Helper()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		t.Fatal(err)
	}
	return cli
}