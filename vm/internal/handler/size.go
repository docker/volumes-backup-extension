package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/docker/volumes-backup-extension/internal/backend"
)

func (h *Handler) VolumeSize(ctx echo.Context) error {
	ctxReq := ctx.Request().Context()
	volumeName := ctx.Param("volume")
	cli, err := h.DockerClient()
	if err != nil {
		return err
	}

	m, err := backend.GetVolumesSize(ctxReq, cli, volumeName)
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, m[volumeName])
}
