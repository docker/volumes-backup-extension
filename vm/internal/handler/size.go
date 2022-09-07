package handler

import (
	"github.com/bugsnag/bugsnag-go/v2"
	"net/http"

	"github.com/docker/volumes-backup-extension/internal/backend"
	"github.com/docker/volumes-backup-extension/internal/log"
	"github.com/labstack/echo"
)

func (h *Handler) VolumeSize(ctx echo.Context) error {
	ctxReq := ctx.Request().Context()
	volumeName := ctx.Param("volume")
	cli, err := h.DockerClient()
	if err != nil {
		log.Error(err)
		_ = bugsnag.Notify(err, ctxReq)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	m, err := backend.GetVolumesSize(ctxReq, cli, volumeName)
	if err != nil {
		log.Error(err)
		_ = bugsnag.Notify(err, ctxReq)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return ctx.JSON(http.StatusOK, m[volumeName])
}
