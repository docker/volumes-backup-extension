package handler

import (
	"github.com/felipecruz91/vackup-docker-extension/internal/backend"
	"github.com/felipecruz91/vackup-docker-extension/internal/log"
	"github.com/labstack/echo"
	"net/http"
)

func (h *Handler) LoadImage(ctx echo.Context) error {
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

	stoppedContainers, err := backend.StopContainersAttachedToVolume(ctx.Request().Context(), h.DockerClient, volumeName)
	if err != nil {
		log.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Load
	err = backend.Load(ctx.Request().Context(), h.DockerClient, volumeName, image)
	if err != nil {
		log.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Start container(s)
	err = backend.StartContainersAttachedToVolume(ctx.Request().Context(), h.DockerClient, stoppedContainers)
	if err != nil {
		log.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return ctx.String(http.StatusOK, "")
}
