package handler

import (
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/docker/distribution/reference"
	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/volumes-backup-extension/internal/backend"
	"github.com/docker/volumes-backup-extension/internal/log"
	"github.com/labstack/echo"
	"github.com/sirupsen/logrus"
)

type PullRequest struct {
	Reference         string `json:"reference"`
	Base64EncodedAuth string `json:"base64EncodedAuth"`
}

// PullVolume pulls a volume from a registry.
// The user must be previously authenticated to the registry with `docker login <registry>`, otherwise it returns 401 StatusUnauthorized.
func (h *Handler) PullVolume(ctx echo.Context) error {
	var request PullRequest
	if err := ctx.Bind(&request); err != nil {
		return err
	}

	ctxReq := ctx.Request().Context()
	volumeName := ctx.Param("volume")
	log.Infof("volumeName: %s", volumeName)
	log.Infof("reference: %s", request.Reference)
	logrus.Infof("received pull request for volume %s\n", volumeName)

	defer func() {
		h.ProgressCache.Lock()
		delete(h.ProgressCache.m, volumeName)
		h.ProgressCache.Unlock()
		_ = backend.TriggerUIRefresh(ctxReq, h.DockerClient)
	}()

	h.ProgressCache.Lock()
	h.ProgressCache.m[volumeName] = "pull"
	h.ProgressCache.Unlock()

	err := backend.TriggerUIRefresh(ctxReq, h.DockerClient)
	if err != nil {
		log.Error(err)
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
		return ctx.String(http.StatusBadRequest, err.Error())
	}
	log.Infof("parsedRef.String(): %s", parsedRef.String())

	// Pull the volume (image) from registry
	log.Infof("Pulling image %s...", parsedRef.String())
	pullResp, err := h.DockerClient.ImagePull(ctxReq, parsedRef.String(), dockertypes.ImagePullOptions{
		RegistryAuth: request.Base64EncodedAuth,
	})

	if err != nil {
		log.Error(err)

		if strings.Contains(err.Error(), "unauthorized: authentication required") {
			return ctx.String(http.StatusUnauthorized, err.Error())
		}

		return ctx.String(http.StatusInternalServerError, err.Error())
	}
	defer pullResp.Close()

	pullRespBytes, err := ioutil.ReadAll(pullResp)
	if err != nil {
		return ctx.String(http.StatusInternalServerError, err.Error())
	}

	for _, line := range strings.Split(string(pullRespBytes), "\n") {
		log.Info(line)
	}

	// Stop container(s)
	stoppedContainers, err := backend.StopContainersAttachedToVolume(ctxReq, h.DockerClient, volumeName)
	if err != nil {
		log.Error(err)
		return ctx.String(http.StatusInternalServerError, err.Error())
	}

	// Load the image into the volume
	log.Infof("Loading image %s into volume %s...", parsedRef.String(), volumeName)
	if err := backend.Load(ctxReq, h.DockerClient, volumeName, parsedRef.String()); err != nil {
		log.Error(err)
		return ctx.String(http.StatusInternalServerError, err.Error())
	}

	// Start container(s)
	err = backend.StartContainersAttachedToVolume(ctxReq, h.DockerClient, stoppedContainers)
	if err != nil {
		log.Error(err)
		return ctx.String(http.StatusInternalServerError, err.Error())
	}

	return ctx.String(http.StatusCreated, "")
}
