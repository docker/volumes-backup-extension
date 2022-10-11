package handler

import (
	"github.com/docker/docker/api/types/filters"
	"net/http"
	"sync"

	"github.com/docker/volumes-backup-extension/internal/backend"
	"github.com/labstack/echo"
)

func (h *Handler) VolumesContainer(ctx echo.Context) error {
	ctxReq := ctx.Request().Context()
	cli, err := h.DockerClient()
	if err != nil {
		return err
	}

	v, err := cli.VolumeList(ctxReq, filters.NewArgs())
	if err != nil {
		return err
	}

	var res = VolumesResponse{
		data: map[string]VolumeData{},
	}

	var wg sync.WaitGroup
	for _, vol := range v.Volumes {
		wg.Add(1)

		go func(volumeName string) {
			defer wg.Done()
			containers := backend.GetContainersForVolume(ctxReq, cli, volumeName, filters.NewArgs())
			res.Lock()
			defer res.Unlock()
			entry, ok := res.data[volumeName]
			if !ok {
				res.data[volumeName] = VolumeData{
					Containers: containers,
				}
				return
			}
			entry.Containers = containers
			res.data[volumeName] = entry
		}(vol.Name)
	}

	wg.Wait()

	return ctx.JSON(http.StatusOK, res.data)
}
