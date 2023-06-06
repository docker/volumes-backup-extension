package handler

import (
	"net/http"
	"sync"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/volume"
	"github.com/labstack/echo/v4"
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
	ctxReq := ctx.Request().Context()

	cli, err := h.DockerClient()
	if err != nil {
		return err
	}

	v, err := cli.VolumeList(ctxReq, volume.ListOptions{Filters: filters.NewArgs()})
	if err != nil {
		return err
	}

	var res = VolumesResponse{
		data: map[string]VolumeData{},
	}

	for _, vol := range v.Volumes {
		res.data[vol.Name] = VolumeData{
			Driver: vol.Driver,
		}
	}

	return ctx.JSON(http.StatusOK, res.data)
}
