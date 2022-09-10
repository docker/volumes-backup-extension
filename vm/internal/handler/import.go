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

func (h *Handler) ImportTarGzFile(ctx echo.Context) error {
	volumeName := ctx.Param("volume")
	path := ctx.QueryParam("path")

	if volumeName == "" {
		return ctx.String(http.StatusBadRequest, "volume is required")
	}
	if path == "" {
		return ctx.String(http.StatusBadRequest, "path is required")
	}

	log.Infof("volumeName: %s", volumeName)
	log.Infof("path: %s", path)

	cli, err := h.DockerClient()
	if err != nil {
		log.Error(err)
		return ctx.String(http.StatusInternalServerError, err.Error())
	}

	defer func() {
		h.ProgressCache.Lock()
		delete(h.ProgressCache.m, volumeName)
		h.ProgressCache.Unlock()
		_ = backend.TriggerUIRefresh(ctx.Request().Context(), cli)
	}()

	h.ProgressCache.Lock()
	h.ProgressCache.m[volumeName] = "import"
	h.ProgressCache.Unlock()

	if err := backend.TriggerUIRefresh(ctx.Request().Context(), cli); err != nil {
		log.Error(err)
		return ctx.String(http.StatusInternalServerError, err.Error())
	}

	// Stop container(s)
	stoppedContainers, err := backend.StopContainersAttachedToVolume(ctx.Request().Context(), cli, volumeName)
	if err != nil {
		log.Error(err)
		return ctx.String(http.StatusInternalServerError, err.Error())
	}

	// Import
	binds := []string{
		volumeName + ":" + "/vackup-volume",
		path + ":" + "/vackup",
	}
	log.Infof("binds: %+v", binds)

	// Ensure the image is present before creating the container
	reader, err := cli.ImagePull(ctx.Request().Context(), internal.BusyboxImage, types.ImagePullOptions{
		Platform: "linux/" + runtime.GOARCH,
	})
	if err != nil {
		return err
	}
	_, err = io.Copy(os.Stdout, reader)
	if err != nil {
		return err
	}

	resp, err := cli.ContainerCreate(ctx.Request().Context(), &container.Config{
		Image:        internal.BusyboxImage,
		AttachStdout: true,
		AttachStderr: true,
		// remove hidden and not-hidden files and folders:
		// ..?* matches all dot-dot files except '..'
		// .[!.]* matches all dot files except '.' and files whose name begins with '..'
		Cmd: []string{"/bin/sh", "-c", "rm -rf /vackup-volume/..?* /vackup-volume/.[!.]* /vackup-volume/* && tar -xvzf /vackup -C /vackup-volume"},
		Labels: map[string]string{
			"com.docker.desktop.extension":        "true",
			"com.docker.desktop.extension.name":   "Volumes Backup & Share",
			"com.docker.compose.project":          "docker_volumes-backup-extension-desktop-extension",
			"com.volumes-backup-extension.action": "import",
			"com.volumes-backup-extension.volume": volumeName,
			"com.volumes-backup-extension.path":   path,
		},
	}, &container.HostConfig{
		Binds: binds,
	}, nil, nil, "")
	if err != nil {
		log.Error(err)
		return ctx.String(http.StatusInternalServerError, err.Error())
	}

	if err := cli.ContainerStart(ctx.Request().Context(), resp.ID, types.ContainerStartOptions{}); err != nil {
		log.Error(err)
		return ctx.String(http.StatusInternalServerError, err.Error())
	}

	var exitCode int64
	statusCh, errCh := cli.ContainerWait(ctx.Request().Context(), resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			log.Error(err)
			return ctx.String(http.StatusInternalServerError, err.Error())
		}
	case status := <-statusCh:
		log.Infof("status: %#+v\n", status)
		exitCode = status.StatusCode
	}

	out, err := cli.ContainerLogs(ctx.Request().Context(), resp.ID, types.ContainerLogsOptions{ShowStdout: true})
	if err != nil {
		log.Error(err)
		return ctx.String(http.StatusInternalServerError, err.Error())
	}

	_, err = stdcopy.StdCopy(os.Stdout, os.Stderr, out)
	if err != nil {
		log.Error(err)
		return ctx.String(http.StatusInternalServerError, err.Error())
	}

	if exitCode != 0 {
		return ctx.String(http.StatusInternalServerError, fmt.Sprintf("container exited with status code %d\n", exitCode))
	}

	err = cli.ContainerRemove(ctx.Request().Context(), resp.ID, types.ContainerRemoveOptions{})
	if err != nil {
		log.Error(err)
		return ctx.String(http.StatusInternalServerError, err.Error())
	}

	// Start container(s)
	err = backend.StartContainersAttachedToVolume(ctx.Request().Context(), cli, stoppedContainers)
	if err != nil {
		log.Error(err)
		return ctx.String(http.StatusInternalServerError, err.Error())
	}

	return ctx.String(http.StatusOK, "")
}
