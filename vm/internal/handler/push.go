package handler

import (
	"encoding/json"
	"fmt"
	"github.com/bugsnag/bugsnag-go/v2"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/docker/distribution/reference"
	dockertypes "github.com/docker/docker/api/types"
	"github.com/felipecruz91/vackup-docker-extension/internal/backend"
	"github.com/felipecruz91/vackup-docker-extension/internal/log"
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
		_ = bugsnag.Notify(err, ctxReq)
		return err
	}

	volumeName := ctx.Param("volume")
	log.Infof("volumeName: %s", volumeName)
	log.Infof("reference: %s", request.Reference)
	log.Infof("received push request for volume %s\n", volumeName)

	defer func() {
		h.ProgressCache.Lock()
		delete(h.ProgressCache.m, volumeName)
		h.ProgressCache.Unlock()
		_ = backend.TriggerUIRefresh(ctxReq, h.DockerClient)
	}()

	h.ProgressCache.Lock()
	h.ProgressCache.m[volumeName] = "push"
	h.ProgressCache.Unlock()

	err := backend.TriggerUIRefresh(ctxReq, h.DockerClient)
	if err != nil {
		log.Error(err)
		_ = bugsnag.Notify(err, ctxReq)
		return ctx.String(http.StatusInternalServerError, err.Error())
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
		_ = bugsnag.Notify(err, ctxReq)
		return ctx.String(http.StatusBadRequest, err.Error())
	}
	log.Infof("parsedRef.String(): %s", parsedRef.String())

	// Stop container(s)
	stoppedContainers, err := backend.StopContainersAttachedToVolume(ctxReq, h.DockerClient, volumeName)
	if err != nil {
		log.Error(err)
		_ = bugsnag.Notify(err, ctxReq)
		return ctx.String(http.StatusInternalServerError, err.Error())
	}

	// Save the content of the volume into an image
	if err := backend.Save(ctxReq, h.DockerClient, volumeName, parsedRef.String()); err != nil {
		log.Error(err)
		_ = bugsnag.Notify(err, ctxReq)
		return ctx.String(http.StatusInternalServerError, err.Error())
	}

	// Push the image to registry
	pushResp, err := h.DockerClient.ImagePush(ctxReq, parsedRef.String(), dockertypes.ImagePushOptions{
		RegistryAuth: request.Base64EncodedAuth,
	})
	if err != nil {
		log.Error(err)
		_ = bugsnag.Notify(err, ctxReq)
		return ctx.String(http.StatusInternalServerError, err.Error())
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
			log.Error(err)
			_ = bugsnag.Notify(fmt.Errorf(pel.Error), ctxReq)
			if pel.Error == "unauthorized: authentication required" || pel.Error == "no basic auth credentials" {
				return ctx.String(http.StatusUnauthorized, pel.Error)
			} else {
				return ctx.String(http.StatusInternalServerError, pel.Error)
			}
		}
	}

	// Start container(s)
	err = backend.StartContainersAttachedToVolume(ctxReq, h.DockerClient, stoppedContainers)
	if err != nil {
		log.Error(err)
		_ = bugsnag.Notify(err, ctxReq)
		return ctx.String(http.StatusInternalServerError, err.Error())
	}

	return ctx.String(http.StatusCreated, "")
}
