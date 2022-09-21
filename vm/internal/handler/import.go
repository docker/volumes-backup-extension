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
	ctxReq := ctx.Request().Context()
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
		return err
	}

	defer func() {
		h.ProgressCache.Lock()
		delete(h.ProgressCache.m, volumeName)
		h.ProgressCache.Unlock()
		_ = backend.TriggerUIRefresh(ctxReq, cli)
	}()

	h.ProgressCache.Lock()
	h.ProgressCache.m[volumeName] = "import"
	h.ProgressCache.Unlock()

	if err := backend.TriggerUIRefresh(ctxReq, cli); err != nil {
		return err
	}

	// Stop container(s)
	stoppedContainers, err := backend.StopContainersAttachedToVolume(ctxReq, cli, volumeName)
	if err != nil {
		return err
	}

	// Import
	binds := []string{
		volumeName + ":" + "/vackup-volume",
		path + ":" + "/vackup",
	}
	log.Infof("binds: %+v", binds)

	// Ensure the image is present before creating the container
	reader, err := cli.ImagePull(ctxReq, internal.AlpineTarZstdImage, types.ImagePullOptions{
		Platform: "linux/" + runtime.GOARCH,
	})
	if err != nil {
		return err
	}
	_, err = io.Copy(os.Stdout, reader)
	if err != nil {
		return err
	}

	// remove hidden and not-hidden files and folders:
	// ..?* matches all dot-dot files except '..'
	// .[!.]* matches all dot files except '.' and files whose name begins with '..'
	rmCmd := "rm -rf /vackup-volume/..?* /vackup-volume/.[!.]* /vackup-volume/*"

	// For backwards compatibility with version 1.0.0 of the extension, we check if the archive contains a root folder named "vackup-volume"
	// If so, we use the "--strip-components=1" flag to decompress the **content** of the root folder (instead of the copying the root folder itself too).
	// tar accepts "-a" to auto-detect the compression format (.tar.gz, .tar.zst or .tar.bz2).
	fullCmd := fmt.Sprintf("%s && if [[ \"$(tar -tf /vackup vackup-volume/)\" ]]; then tar -axvf /vackup --strip-components=1 -C /vackup-volume; else tar -axvf /vackup -C /vackup-volume; fi", rmCmd)
	log.Infof("fullCmd: %s", fullCmd)

	resp, err := cli.ContainerCreate(ctxReq, &container.Config{
		Image:        internal.AlpineTarZstdImage,
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          []string{"/bin/sh", "-c", fullCmd},
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

	out, err := cli.ContainerLogs(ctxReq, resp.ID, types.ContainerLogsOptions{ShowStdout: true})
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

	return ctx.String(http.StatusOK, "")
}
