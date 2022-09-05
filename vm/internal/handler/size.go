package handler

import (
	"net/http"

	"github.com/docker/volumes-backup-extension/internal/backend"
	"github.com/labstack/echo"
)

func (h *Handler) VolumeSize(ctx echo.Context) error {
	volumeName := ctx.Param("volume")

	m := backend.GetVolumesSize(ctx.Request().Context(), h.DockerClient, volumeName)

	return ctx.JSON(http.StatusOK, m[volumeName])
}
