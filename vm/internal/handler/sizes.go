package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/docker/volumes-backup-extension/internal/backend"
)

func (h *Handler) VolumesSize(ctx echo.Context) error {
	cli, err := h.DockerClient()
	if err != nil {
		return err
	}

	m, err := backend.GetVolumesSize(ctx.Request().Context(), cli, "*")
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, m)
}
