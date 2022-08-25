package handler

import (
	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/felipecruz91/vackup-docker-extension/internal/log"
	"github.com/labstack/echo"
	"net/http"
	"sync"
)

type ProgressCache struct {
	sync.RWMutex
	m map[string]string // map of volumes and actions, e.g. m["vol-1"] = "export"
}

// ActionsInProgress retrieves the current action (i.e. export, import, save or load) that is running for every volume.
func (h *Handler) ActionsInProgress(ctx echo.Context) error {
	ctxReq := ctx.Request().Context()
	containers, err := h.DockerClient.ContainerList(ctxReq, dockertypes.ContainerListOptions{
		Quiet: true,
		All:   true,
		Filters: filters.NewArgs(
			filters.Arg("label", "com.docker.compose.project=docker_volumes-backup-extension-desktop-extension"),
			filters.Arg("status", "running")),
	})
	if err != nil {
		log.Error(err)
		return ctx.String(http.StatusInternalServerError, err.Error())
	}

	// TODO: go routines
	for _, c := range containers {
		cJSON, err := h.DockerClient.ContainerInspect(ctxReq, c.ID)
		if err != nil {
			continue
		}

		var action string
		var volume string
		for key, value := range cJSON.Config.Labels {
			switch key {
			case "com.volumes-backup-extension.action":
				action = value
			case "com.volumes-backup-extension.volume":
				volume = value
			}
		}

		h.ProgressCache.Lock()
		h.ProgressCache.m[volume] = action
		h.ProgressCache.Unlock()
	}

	log.Infof("progress cache: %+v", h.ProgressCache.m)

	return ctx.JSON(http.StatusOK, h.ProgressCache.m)
}
