package handler

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/volumes-backup-extension/internal"
	"github.com/docker/volumes-backup-extension/internal/backend"
	"github.com/docker/volumes-backup-extension/internal/log"
	"github.com/labstack/echo"
)

func (h *Handler) ExportVolume(ctx echo.Context) error {
	ctxReq := ctx.Request().Context()
	volumeName := ctx.Param("volume")
	path := ctx.QueryParam("path")
	fileName := ctx.QueryParam("fileName")

	if volumeName == "" {
		return ctx.String(http.StatusBadRequest, "volume is required")
	}
	if path == "" {
		return ctx.String(http.StatusBadRequest, "path is required")
	}
	if fileName == "" {
		return ctx.String(http.StatusBadRequest, "path is required")
	}

	log.Infof("volumeName: %s", volumeName)
	log.Infof("path: %s", path)
	log.Infof("fileName: %s", fileName)

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
	h.ProgressCache.m[volumeName] = "export"
	h.ProgressCache.Unlock()

	if err := backend.TriggerUIRefresh(ctxReq, cli); err != nil {
		return err
	}

	// Stop container(s)
	stoppedContainers, err := backend.StopContainersAttachedToVolume(ctxReq, cli, volumeName)
	if err != nil {
		return err
	}

	var compressProgram string
	tarOpts := "-cvf"

	fileExt := filepath.Ext(fileName)
	log.Infof("fileExt: %s", fileExt)

	switch fileExt {
	case ".gz":
		//compressProgram = "gzip" // TODO: use pigz (parallel implementation of gzip)
		tarOpts = tarOpts[1:] + "z" // remove "-" from first tarOptos, specify "z" to indicate gzip compression
	case ".zst":
		compressProgram = "zstdmt" // zstdmt is equivalent to zstd -T0 (attempt to detect and use the number of physical CPU cores)
	case ".bz2":
		compressProgram = "bzip2" // TODO: install bzip2 in AlpineTarZstdImage
		tarOpts += "j"            // bzip compression
	default:
		compressProgram = ""
	}

	// Export
	cmd := []string{"tar"}

	if compressProgram != "" {
		cmd = append(cmd, "-I", compressProgram)
	}

	cmd = append(cmd,
		tarOpts,
		"/vackup"+"/"+filepath.Base(fileName), // the .tar.zst file
		"-C",             // -C is used to not include the parent directory
		"/vackup-volume", // the directory where the files to compress are
		".")

	cmdJoined := strings.Join(cmd, " ")
	log.Infof("cmdJoined: %s", cmdJoined)

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

	resp, err := cli.ContainerCreate(ctxReq, &container.Config{
		Image:        internal.AlpineTarZstdImage,
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          []string{"/bin/sh", "-c", cmdJoined},
		User:         "root",
		Labels: map[string]string{
			"com.docker.desktop.extension":          "true",
			"com.docker.desktop.extension.name":     "Volumes Backup & Share",
			"com.docker.compose.project":            "docker_volumes-backup-extension-desktop-extension",
			"com.volumes-backup-extension.action":   "export",
			"com.volumes-backup-extension.volume":   volumeName,
			"com.volumes-backup-extension.path":     path,
			"com.volumes-backup-extension.fileName": fileName,
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
