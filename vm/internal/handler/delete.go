package handler

import (
	"net/http"

	"github.com/docker/volumes-backup-extension/internal/backend"
	"github.com/docker/volumes-backup-extension/internal/log"
	"github.com/labstack/echo"
)

func (h *Handler) DeleteVolume(ctx echo.Context) error {
	volumeName := ctx.Param("volume")

	if volumeName == "" {
		return ctx.String(http.StatusBadRequest, "volume is required")
	}

	log.Infof("volumeName: %s", volumeName)

	defer func() {
		h.ProgressCache.Lock()
		delete(h.ProgressCache.m, volumeName)
		h.ProgressCache.Unlock()
		_ = backend.TriggerUIRefresh(ctx.Request().Context(), h.DockerClient)
	}()

	h.ProgressCache.Lock()
	h.ProgressCache.m[volumeName] = "delete"
	h.ProgressCache.Unlock()

	err := backend.TriggerUIRefresh(ctx.Request().Context(), h.DockerClient)
	if err != nil {
		log.Error(err)
		return ctx.String(http.StatusInternalServerError, err.Error())
	}

	// Delete volume
	err = h.DockerClient.VolumeRemove(ctx.Request().Context(), volumeName, true)
	if err != nil {
		log.Error(err)
		return ctx.String(http.StatusInternalServerError, err.Error())
	}

	return ctx.String(http.StatusNoContent, "")
}
