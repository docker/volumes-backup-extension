package handler

import (
	"encoding/json"
	"github.com/docker/distribution/reference"
	dockertypes "github.com/docker/docker/api/types"
	"github.com/felipecruz91/vackup-docker-extension/internal/backend"
	"github.com/felipecruz91/vackup-docker-extension/internal/log"
	"github.com/labstack/echo"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"strings"
)

type PullRequest struct {
	Reference string `json:"reference"`
}

type PullErrorLine struct {
	ErrorDetail ErrorDetail `json:"errorDetail"`
	Error       string      `json:"error"`
}

// PullVolume pulls a volume from a registry.
// The user must be previously authenticated to the registry with `docker login <registry>`, otherwise it returns 401 StatusUnauthorized.
func (h *Handler) PullVolume(ctx echo.Context) error {
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

	// Push the image to registry
	encodedAuth := ctx.Request().Header.Get("X-Registry-Auth")
	if encodedAuth == "" {
		encodedAuth = "Cg==" // from running: echo "" | base64
	}
	pullResp, err := h.DockerClient.ImagePull(ctxReq, parsedRef.String(), dockertypes.ImagePullOptions{
		RegistryAuth: encodedAuth,
	})
	if err != nil {
		log.Error(err)
		return ctx.String(http.StatusInternalServerError, err.Error())
	}
	defer pullResp.Close()

	response, err := ioutil.ReadAll(pullResp)

	for _, line := range strings.Split(string(response), "\n") {
		log.Info(line)

		if !strings.Contains(line, "error") {
			continue
		}

		pel := PushErrorLine{}
		if err := json.Unmarshal([]byte(line), &pel); err == nil {
			// TODO: double check
			// the image pull had an error, e.g:
			// {"errorDetail":{"message":"unauthorized: authentication required"},"error":"unauthorized: authentication required"}
			// or
			// {"errorDetail":{"message":"no basic auth credentials"},"error":"no basic auth credentials"}
			log.Error(err)
			if pel.Error == "unauthorized: authentication required" || pel.Error == "no basic auth credentials" {
				return ctx.String(http.StatusUnauthorized, pel.Error)
			} else {
				return ctx.String(http.StatusInternalServerError, pel.Error)
			}
		}
	}

	// Load the image into the volume
	if err := backend.Load(ctxReq, h.DockerClient, volumeName, parsedRef.String()); err != nil {
		log.Error(err)
		return ctx.String(http.StatusInternalServerError, err.Error())
	}

	return ctx.String(http.StatusCreated, "")
}
