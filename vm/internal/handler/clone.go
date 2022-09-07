package handler

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/volumes-backup-extension/internal"
	"github.com/docker/volumes-backup-extension/internal/backend"
	"github.com/docker/volumes-backup-extension/internal/log"
	"github.com/labstack/echo"
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
	h.ProgressCache.m[volumeName] = "clone"
	h.ProgressCache.Unlock()

	if err := backend.TriggerUIRefresh(ctxReq, cli); err != nil {
		return err
	}

	// Stop container(s)
	stoppedContainers, err := backend.StopContainersAttachedToVolume(ctxReq, cli, volumeName)
	if err != nil {
		return err
	}

	// Ensure the image is present before creating the container
	reader, err := cli.ImagePull(ctxReq, internal.BusyboxImage, types.ImagePullOptions{
		Platform: "linux/" + runtime.GOARCH,
	})
	if err != nil {
		return err
	}
	_, err = io.Copy(os.Stdout, reader)
	if err != nil {
		return err
	}

	// Clone
	resp, err := cli.ContainerCreate(ctxReq, &container.Config{
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
		return err
	}

	if err := cli.ContainerStart(ctxReq, resp.ID, types.ContainerStartOptions{}); err != nil {
		return err
	}

	var exitCode int64
	statusCh, errCh := cli.ContainerWait(ctxReq, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return err
		}
	case status := <-statusCh:
		log.Infof("status: %#+v\n", status)
		exitCode = status.StatusCode
	}

	out, err := cli.ContainerLogs(ctxReq, resp.ID, types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true})
	if err != nil {
		return err
	}

	_, err = stdcopy.StdCopy(os.Stdout, os.Stderr, out)
	if err != nil {
		return err
	}

	if exitCode != 0 {
		return ctx.String(http.StatusInternalServerError, fmt.Sprintf("container exited with status code %d\n", exitCode))
	}

	err = cli.ContainerRemove(ctxReq, resp.ID, types.ContainerRemoveOptions{})
	if err != nil {
		return err
	}

	// Start container(s)
	err = backend.StartContainersAttachedToVolume(ctxReq, cli, stoppedContainers)
	if err != nil {
		return err
	}

	return ctx.String(http.StatusCreated, "")
}
