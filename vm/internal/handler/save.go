package handler

import (
	"net/http"

	"github.com/docker/volumes-backup-extension/internal/backend"
	"github.com/docker/volumes-backup-extension/internal/log"
	"github.com/labstack/echo"
)

func (h *Handler) SaveVolume(ctx echo.Context) error {
	ctxReq := ctx.Request().Context()
	volumeName := ctx.Param("volume")
	image := ctx.QueryParam("image")

	if volumeName == "" {
		return ctx.String(http.StatusBadRequest, "volume is required")
	}
	if image == "" {
		return ctx.String(http.StatusBadRequest, "image is required")
	}

	log.Infof("volumeName: %s", volumeName)
	log.Infof("image: %s", image)

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
	h.ProgressCache.m[volumeName] = "save"
	h.ProgressCache.Unlock()

	err = backend.TriggerUIRefresh(ctxReq, cli)
	if err != nil {
		return err
	}

	// Stop container(s)
	stoppedContainers, err := backend.StopRunningContainersAttachedToVolume(ctxReq, cli, volumeName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Save volume into an image
	if err := backend.Save(ctxReq, cli, volumeName, image); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Start container(s)
	err = backend.StartContainersByName(ctxReq, cli, stoppedContainers)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return ctx.String(http.StatusCreated, "")
}
