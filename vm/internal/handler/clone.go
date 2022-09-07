package handler

import (
	"fmt"
	"github.com/bugsnag/bugsnag-go/v2"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/felipecruz91/vackup-docker-extension/internal"
	"github.com/felipecruz91/vackup-docker-extension/internal/backend"
	"github.com/felipecruz91/vackup-docker-extension/internal/log"
	"github.com/labstack/echo"
	"io"
	"net/http"
	"os"
	"runtime"
)

func (h *Handler) CloneVolume(ctx echo.Context) error {
	ctxReq := ctx.Request().Context()
	volumeName := ctx.Param("volume")
	destVolume := ctx.QueryParam("destVolume")

	if volumeName == "" {
		return ctx.String(http.StatusBadRequest, "volume is required")
	}
	if destVolume == "" {
		return ctx.String(http.StatusBadRequest, "destVolume is required")
	}

	log.Infof("volumeName: %s", volumeName)
	log.Infof("destVolume: %s", destVolume)

	defer func() {
		h.ProgressCache.Lock()
		delete(h.ProgressCache.m, volumeName)
		h.ProgressCache.Unlock()
		_ = backend.TriggerUIRefresh(ctxReq, h.DockerClient)
	}()

	h.ProgressCache.Lock()
	h.ProgressCache.m[volumeName] = "clone"
	h.ProgressCache.Unlock()

	err := backend.TriggerUIRefresh(ctxReq, h.DockerClient)
	if err != nil {
		log.Error(err)
		_ = bugsnag.Notify(err, ctxReq)
		return ctx.String(http.StatusInternalServerError, err.Error())
	}

	// Stop container(s)
	stoppedContainers, err := backend.StopContainersAttachedToVolume(ctxReq, h.DockerClient, volumeName)
	if err != nil {
		log.Error(err)
		_ = bugsnag.Notify(err, ctxReq)
		return ctx.String(http.StatusInternalServerError, err.Error())
	}

	// Ensure the image is present before creating the container
	reader, err := h.DockerClient.ImagePull(ctxReq, internal.BusyboxImage, types.ImagePullOptions{
		Platform: "linux/" + runtime.GOARCH,
	})
	if err != nil {
		_ = bugsnag.Notify(err, ctxReq)
		return err
	}
	_, err = io.Copy(os.Stdout, reader)
	if err != nil {
		_ = bugsnag.Notify(err, ctxReq)
		return err
	}

	// Clone
	resp, err := h.DockerClient.ContainerCreate(ctxReq, &container.Config{
		Image:        internal.BusyboxImage,
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          []string{"/bin/sh", "-c", "cd /from ; cp -av . /to"},
		User:         "root",
		Labels: map[string]string{
			"com.docker.desktop.extension":                    "true",
			"com.docker.desktop.extension.name":               "Volumes Backup & Share",
			"com.docker.compose.project":                      "docker_volumes-backup-extension-desktop-extension",
			"com.volumes-backup-extension.action":             "clone",
			"com.volumes-backup-extension.volume":             volumeName,
			"com.volumes-backup-extension.destination-volume": destVolume,
		},
	}, &container.HostConfig{
		Binds: []string{
			volumeName + ":" + "/from",
			destVolume + ":" + "/to",
		},
	}, nil, nil, "")
	if err != nil {
		log.Error(err)
		_ = bugsnag.Notify(err, ctxReq)
		return ctx.String(http.StatusInternalServerError, err.Error())
	}

	if err := h.DockerClient.ContainerStart(ctxReq, resp.ID, types.ContainerStartOptions{}); err != nil {
		log.Error(err)
		_ = bugsnag.Notify(err, ctxReq)
		return ctx.String(http.StatusInternalServerError, err.Error())
	}

	var exitCode int64
	statusCh, errCh := h.DockerClient.ContainerWait(ctxReq, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			log.Error(err)
			_ = bugsnag.Notify(err, ctxReq)
			return ctx.String(http.StatusInternalServerError, err.Error())
		}
	case status := <-statusCh:
		log.Infof("status: %#+v\n", status)
		exitCode = status.StatusCode
	}

	out, err := h.DockerClient.ContainerLogs(ctxReq, resp.ID, types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true})
	if err != nil {
		log.Error(err)
		_ = bugsnag.Notify(err, ctxReq)
		return ctx.String(http.StatusInternalServerError, err.Error())
	}

	_, err = stdcopy.StdCopy(os.Stdout, os.Stderr, out)
	if err != nil {
		log.Error(err)
		_ = bugsnag.Notify(err, ctxReq)
		return ctx.String(http.StatusInternalServerError, err.Error())
	}

	if exitCode != 0 {
		return ctx.String(http.StatusInternalServerError, fmt.Sprintf("container exited with status code %d\n", exitCode))
	}

	err = h.DockerClient.ContainerRemove(ctxReq, resp.ID, types.ContainerRemoveOptions{})
	if err != nil {
		log.Error(err)
		_ = bugsnag.Notify(err, ctxReq)
		return ctx.String(http.StatusInternalServerError, err.Error())
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
