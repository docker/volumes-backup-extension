package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	volumetypes "github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

func TestDeleteVolume(t *testing.T) {
	var containerID string
	volume := "dc0c85c49196932194dcf34e2a1280dcd4a8c46653c407c4e06845ffad3109ba"
	cli := setupDockerClient(t)

	defer func() {
		_ = cli.ContainerRemove(context.Background(), containerID, types.ContainerRemoveOptions{
			Force: true,
		})
		_ = cli.VolumeRemove(context.Background(), volume, true)
	}()

	// Setup
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/volumes/:volume/delete")
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

	// Delete volume
	err = h.DeleteVolume(c)
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, rec.Code)

	// Check the volume has been deleted
	clonedVolumeResp, err := cli.VolumeList(context.Background(), filters.NewArgs(filters.Arg("name", volume)))
	if err != nil {
		t.Fatal(err)
	}
	require.Len(t, clonedVolumeResp.Volumes, 0)
}
