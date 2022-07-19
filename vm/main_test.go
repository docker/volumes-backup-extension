package main

import (
	"context"
	"encoding/json"
	volumetypes "github.com/docker/docker/api/types/volume"
	"github.com/labstack/echo"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
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
	c.SetPath("/volumes/:volume/size")
	c.SetParamNames("volume")
	c.SetParamValues(volume)

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
