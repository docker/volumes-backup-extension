package handler

import (
	"net/http"

	"github.com/docker/volumes-backup-extension/internal/backend"
	"github.com/docker/volumes-backup-extension/internal/log"
	"github.com/labstack/echo"
)

func (h *Handler) DeleteVolume(ctx echo.Context) error {
	ctxReq := ctx.Request().Context()
	volumeName := ctx.Param("volume")

	if volumeName == "" {
		return ctx.String(http.StatusBadRequest, "volume is required")
	}

	log.Infof("volumeName: %s", volumeName)

	cli, err := h.DockerClient()
	if err != nil {
		return err
	}

	defer func() {
		h.ProgressCache.Lock()
		delete(h.ProgressCache.m, volumeName)
		h.ProgressCache.Unlock()
		_ = backend.TriggerUIRefresh(ctxReq, cli)
	}()

	h.ProgressCache.Lock()
	h.ProgressCache.m[volumeName] = "delete"
	h.ProgressCache.Unlock()

	if err := backend.TriggerUIRefresh(ctxReq, cli); err != nil {
		return err
	}

	// Delete volume
	err = cli.VolumeRemove(ctxReq, volumeName, true)
	if err != nil {
		return err
	}

	return ctx.String(http.StatusNoContent, "")
}
