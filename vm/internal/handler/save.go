package handler

import (
	"github.com/bugsnag/bugsnag-go/v2"
	"github.com/felipecruz91/vackup-docker-extension/internal/backend"
	"github.com/felipecruz91/vackup-docker-extension/internal/log"
	"github.com/labstack/echo"
	"net/http"
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

	defer func() {
		h.ProgressCache.Lock()
		delete(h.ProgressCache.m, volumeName)
		h.ProgressCache.Unlock()
		_ = backend.TriggerUIRefresh(ctxReq, h.DockerClient)
	}()

	h.ProgressCache.Lock()
	h.ProgressCache.m[volumeName] = "save"
	h.ProgressCache.Unlock()

	err := backend.TriggerUIRefresh(ctxReq, h.DockerClient)
	if err != nil {
		log.Error(err)
		_ = bugsnag.Notify(err, ctxReq)
		return ctx.String(http.StatusInternalServerError, err.Error())
	}

	// Stop container(s)
	stoppedContainers, err := backend.StopContainersAttachedToVolume(ctxReq, h.DockerClient, volumeName)
	if err != nil {
		log.Error(err)
		_ = bugsnag.Notify(err, ctxReq)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Save volume into an image
	if err := backend.Save(ctxReq, h.DockerClient, volumeName, image); err != nil {
		log.Error(err)
		_ = bugsnag.Notify(err, ctxReq)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Start container(s)
	err = backend.StartContainersAttachedToVolume(ctxReq, h.DockerClient, stoppedContainers)
	if err != nil {
		log.Error(err)
		_ = bugsnag.Notify(err, ctxReq)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return ctx.String(http.StatusCreated, "")
}
