package handler

import (
	"net/http"

	"github.com/bugsnag/bugsnag-go/v2"
	"github.com/docker/volumes-backup-extension/internal/backend"
	"github.com/docker/volumes-backup-extension/internal/log"
	"github.com/labstack/echo"
)

func (h *Handler) LoadImage(ctx echo.Context) error {
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
	h.ProgressCache.m[volumeName] = "load"
	h.ProgressCache.Unlock()

	if err := backend.TriggerUIRefresh(ctxReq, cli); err != nil {
		return err
	}

	stoppedContainers, err := backend.StopContainersAttachedToVolume(ctxReq, cli, volumeName)
	if err != nil {
		return err
	}

	// Load
	err = backend.Load(ctxReq, cli, volumeName, image)
	if err != nil {
		return err
	}

	// Start container(s)
	err = backend.StartContainersAttachedToVolume(ctxReq, cli, stoppedContainers)
	if err != nil {
		log.Error(err)
		_ = bugsnag.Notify(err, ctxReq)
		return err
	}

	return ctx.String(http.StatusOK, "")
}
