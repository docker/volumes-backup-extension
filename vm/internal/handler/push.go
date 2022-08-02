package handler

import (
	"bytes"
	"github.com/docker/distribution/reference"
	dockertypes "github.com/docker/docker/api/types"
	"github.com/felipecruz91/vackup-docker-extension/internal/backend"
	"github.com/felipecruz91/vackup-docker-extension/internal/log"
	"github.com/labstack/echo"
	"github.com/sirupsen/logrus"
	"net/http"
)

type PushRequest struct {
	Reference string `json:"reference"`
}

// PushVolume pushes a volume to a registry.
// The user must be previously authenticated to the registry with `docker login <registry>`.
func (h *Handler) PushVolume(ctx echo.Context) error {
	var request PushRequest
	if err := ctx.Bind(&request); err != nil {
		return err
	}

	ctxReq := ctx.Request().Context()
	volumeName := ctx.Param("volume")
	log.Infof("volumeName: %s", volumeName)
	log.Infof("reference: %s", request.Reference)
	logrus.Infof("received push request for volume %s\n", volumeName)

	if volumeName == "" {
		return ctx.String(http.StatusBadRequest, "volume is required")
	}

	parsedRef, err := reference.ParseAnyReference(request.Reference)
	if err != nil {
		return ctx.String(http.StatusBadRequest, err.Error())
	}
	log.Infof("parsedRef.String(): %s", parsedRef.String())

	// Save the content of the volume into an image
	if err := backend.Save(ctxReq, h.DockerClient, volumeName, parsedRef.String()); err != nil {
		log.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Push the image to registry
	encodedAuth := ctx.Request().Header.Get("X-Registry-Auth")
	if encodedAuth == "" {
		encodedAuth = "Cg==" // from running: echo "" | base64
	}
	r, err := h.DockerClient.ImagePush(ctxReq, parsedRef.String(), dockertypes.ImagePushOptions{
		RegistryAuth: encodedAuth,
	})
	if err != nil {
		log.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	defer r.Close()

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(r)
	if err != nil {
		log.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	log.Info(buf.String())

	return ctx.String(http.StatusCreated, "")
}
