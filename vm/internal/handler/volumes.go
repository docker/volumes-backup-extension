package handler

import (
	"net/http"
	"sync"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/volumes-backup-extension/internal/log"
	"github.com/labstack/echo"
)

type VolumesResponse struct {
	sync.RWMutex
	data map[string]VolumeData
}

type VolumeData struct {
	Driver     string
	Size       int64
	SizeHuman  string
	Containers []string
}

func (h *Handler) Volumes(ctx echo.Context) error {
	cli, err := h.DockerClient()
	if err != nil {
		log.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	v, err := cli.VolumeList(ctx.Request().Context(), filters.NewArgs())
	if err != nil {
		log.Error(err)
	}

	var res = VolumesResponse{
		data: map[string]VolumeData{},
	}

	for _, vol := range v.Volumes {
		res.data[vol.Name] = VolumeData{
			Driver:     vol.Driver,
			Size:       -1,   // set to `-1` if the value is not available.
			SizeHuman:  "-1", // set to `-1` if the value is not available.
			Containers: nil,  // set to `nil` if the value is not available.
		}
	}

	return ctx.JSON(http.StatusOK, res.data)
}
