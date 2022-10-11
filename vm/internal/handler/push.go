package handler

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/docker/distribution/reference"
	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/volumes-backup-extension/internal/backend"
	"github.com/docker/volumes-backup-extension/internal/log"
	"github.com/labstack/echo"
)

type PushRequest struct {
	Reference         string `json:"reference"`
	Base64EncodedAuth string `json:"base64EncodedAuth"`
}

type PushErrorLine struct {
	ErrorDetail ErrorDetail `json:"errorDetail"`
	Error       string      `json:"error"`
}
type ErrorDetail struct {
	Message string `json:"message"`
}

// PushVolume pushes a volume to a registry.
// The user must be previously authenticated to the registry with `docker login <registry>`, otherwise it returns 401 StatusUnauthorized.
func (h *Handler) PushVolume(ctx echo.Context) error {
	ctxReq := ctx.Request().Context()

	var request PushRequest
	if err := ctx.Bind(&request); err != nil {
		return err
	}

	volumeName := ctx.Param("volume")
	log.Infof("volumeName: %s", volumeName)
	log.Infof("reference: %s", request.Reference)
	log.Infof("received push request for volume %s\n", volumeName)

	cli, err := h.DockerClient()
	if err != nil {
		return err
	}
	defer func() {
		h.ProgressCache.Lock()
		delete(h.ProgressCache.m, volumeName)
		h.ProgressCache.Unlock()
		_ = backend.TriggerUIRefresh(ctxReq, cli)
	}()

	h.ProgressCache.Lock()
	h.ProgressCache.m[volumeName] = "push"
	h.ProgressCache.Unlock()

	err = backend.TriggerUIRefresh(ctxReq, cli)
	if err != nil {
		return err
	}

	// To provide backwards compatibility with older versions of Docker Desktop,
	// we're passing the encoded auth in the body of the request instead of in the headers.
	// encodedAuth := ctx.Request().Header.Get("X-Registry-Auth")
	if request.Base64EncodedAuth == "" {
		request.Base64EncodedAuth = "Cg==" // from running: echo "" | base64
	}

	if volumeName == "" {
		return ctx.String(http.StatusBadRequest, "volume is required")
	}

	parsedRef, err := reference.ParseAnyReference(request.Reference)
	if err != nil {
		return ctx.String(http.StatusBadRequest, err.Error())
	}
	log.Infof("parsedRef.String(): %s", parsedRef.String())

	// Stop container(s)
	stoppedContainers, err := backend.StopRunningContainersAttachedToVolume(ctxReq, cli, volumeName)
	if err != nil {
		return err
	}

	// Save the content of the volume into an image
	if err := backend.Save(ctxReq, cli, volumeName, parsedRef.String()); err != nil {
		return err
	}

	// Push the image to registry
	pushResp, err := cli.ImagePush(ctxReq, parsedRef.String(), dockertypes.ImagePushOptions{
		RegistryAuth: request.Base64EncodedAuth,
	})
	if err != nil {
		return err
	}
	defer pushResp.Close()

	response, err := ioutil.ReadAll(pushResp)

	for _, line := range strings.Split(string(response), "\n") {
		log.Info(line)

		if !strings.Contains(line, "error") {
			continue
		}

		pel := PushErrorLine{}
		if err := json.Unmarshal([]byte(line), &pel); err == nil {
			// the image push had an error, e.g:
			// {"errorDetail":{"message":"unauthorized: authentication required"},"error":"unauthorized: authentication required"}
			// or
			// {"errorDetail":{"message":"no basic auth credentials"},"error":"no basic auth credentials"}
			if pel.Error == "unauthorized: authentication required" || pel.Error == "no basic auth credentials" {
				return ctx.String(http.StatusUnauthorized, pel.Error)
			} else {
				return ctx.String(http.StatusInternalServerError, pel.Error)
			}
		}
	}

	// Start container(s)
	err = backend.StartContainersByName(ctxReq, cli, stoppedContainers)
	if err != nil {
		return err
	}

	return ctx.String(http.StatusCreated, "")
}
