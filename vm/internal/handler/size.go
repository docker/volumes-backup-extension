package handler

import (
	"net/http"

	"github.com/docker/volumes-backup-extension/internal/backend"
	"github.com/docker/volumes-backup-extension/internal/log"
	"github.com/labstack/echo"
)

func (h *Handler) VolumeSize(ctx echo.Context) error {
	volumeName := ctx.Param("volume")
	cli, err := h.DockerClient()
	if err != nil {
		log.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	m := backend.GetVolumesSize(ctx.Request().Context(), cli, volumeName)

	return ctx.JSON(http.StatusOK, m[volumeName])
}
