package handler

import (
	"github.com/bugsnag/bugsnag-go/v2"
	"github.com/felipecruz91/vackup-docker-extension/internal/backend"
	"github.com/felipecruz91/vackup-docker-extension/internal/log"
	"github.com/labstack/echo"
	"net/http"
)

func (h *Handler) DeleteVolume(ctx echo.Context) error {
	ctxReq := ctx.Request().Context()
	volumeName := ctx.Param("volume")

	if volumeName == "" {
		return ctx.String(http.StatusBadRequest, "volume is required")
	}

	log.Infof("volumeName: %s", volumeName)

	defer func() {
		h.ProgressCache.Lock()
		delete(h.ProgressCache.m, volumeName)
		h.ProgressCache.Unlock()
		_ = backend.TriggerUIRefresh(ctxReq, h.DockerClient)
	}()

	h.ProgressCache.Lock()
	h.ProgressCache.m[volumeName] = "delete"
	h.ProgressCache.Unlock()

	err := backend.TriggerUIRefresh(ctxReq, h.DockerClient)
	if err != nil {
		log.Error(err)
		_ = bugsnag.Notify(err, ctxReq)
		return ctx.String(http.StatusInternalServerError, err.Error())
	}

	// Delete volume
	err = h.DockerClient.VolumeRemove(ctxReq, volumeName, true)
	if err != nil {
		log.Error(err)
		_ = bugsnag.Notify(err, ctxReq)
		return ctx.String(http.StatusInternalServerError, err.Error())
	}

	return ctx.String(http.StatusNoContent, "")
}
