package handler

import (
	"github.com/felipecruz91/vackup-docker-extension/internal/backend"
	"github.com/labstack/echo"
	"github.com/sirupsen/logrus"
	"net/http"
	"time"
)

func (h *Handler) VolumeSize(ctx echo.Context) error {
	start := time.Now()

	volumeName := ctx.Param("volume")
	m := backend.GetVolumeSize(ctx.Request().Context(), h.DockerClient, volumeName)

	logrus.Infof("/volumeSize took %s", time.Since(start))
	return ctx.JSON(http.StatusOK, m[volumeName])
}
