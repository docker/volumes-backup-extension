package handler

import (
	"context"
	"net/http"
	"sync"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/volumes-backup-extension/internal/backend"
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

	var wg sync.WaitGroup
	// Calculating the volume size by spinning a container that execs "du " **per volume** is too time-consuming.
	// To reduce the time it takes, we get the volumes size by running only one container that execs "du"
	// into the /var/lib/docker/volumes inside the VM.
	volumesSize := backend.GetVolumesSize(ctx.Request().Context(), cli, "*")
	res.Lock()
	for k, v := range volumesSize {
		entry, ok := res.data[k]
		if !ok {
			res.data[k] = VolumeData{
				Size:      v.Bytes,
				SizeHuman: v.Human,
			}
			continue
		}
		entry.Size = v.Bytes
		entry.SizeHuman = v.Human
		res.data[k] = entry
	}
	res.Unlock()

	for _, vol := range v.Volumes {
		wg.Add(2)
		go func(volumeName string) {
			defer wg.Done()
			driver := backend.GetVolumeDriver(context.Background(), cli, volumeName) // TODO: use request context
			res.Lock()
			defer res.Unlock()
			entry, ok := res.data[volumeName]
			if !ok {
				res.data[volumeName] = VolumeData{
					Driver: driver,
				}
				return
			}
			entry.Driver = driver
			res.data[volumeName] = entry
		}(vol.Name)

		go func(volumeName string) {
			defer wg.Done()
			containers := backend.GetContainersForVolume(context.Background(), cli, volumeName) // TODO: use request context
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
