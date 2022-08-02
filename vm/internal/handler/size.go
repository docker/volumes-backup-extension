package handler

import (
	"github.com/felipecruz91/vackup-docker-extension/internal/backend"
	"github.com/labstack/echo"
	"net/http"
)

func (h *Handler) VolumeSize(ctx echo.Context) error {
	volumeName := ctx.Param("volume")

	m := backend.GetVolumesSize(ctx.Request().Context(), h.DockerClient, volumeName)

	return ctx.JSON(http.StatusOK, m[volumeName])
}
