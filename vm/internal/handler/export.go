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
	"github.com/labstack/echo/v4"

	"github.com/docker/volumes-backup-extension/internal"
	"github.com/docker/volumes-backup-extension/internal/backend"
	"github.com/docker/volumes-backup-extension/internal/log"
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

	// Check if the target file already exists
	targetFilePath := filepath.Join(path, fileName)
	if _, err := os.Stat(targetFilePath); err == nil {
		// File exists, prompt for confirmation
		fmt.Println("A file with the same name already exists in the target directory.")
		fmt.Println("Do you want to:")
		fmt.Println("1. Overwrite the existing file (O)")
		fmt.Println("2. Keep both files (K)")
		fmt.Println("3. Cancel the export (C)")

		var userInput string
		fmt.Scanln(&userInput)

		switch strings.ToLower(userInput) {
		case "o": // Overwrite
			// Continue with export
		case "k": // Keep both
			// Modify fileName to avoid overwriting
			fileName = fileName + "(1)"
			targetFilePath = filepath.Join(path, fileName)
		case "c": // Cancel
			return ctx.String(http.StatusOK, "Export canceled.")
		default:
			return ctx.String(http.StatusBadRequest, "Invalid input. Please choose a valid option.")
		}
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
	stoppedContainers, err := backend.StopRunningContainersAttachedToVolume(ctxReq, cli, volumeName)
	if err != nil {
		return err
	}

	var compressProgram string
	tarOpts := "-cvf"

	fileExt := filepath.Ext(fileName)
	log.Infof("fileExt: %s", fileExt)

	switch fileExt {
	case ".gz":
		compressProgram = "\"pigz -6 -k\"" // pigz (parallel implementation of gzip), use -6 as the default compression level (-1 is fastest, -9 is best), "-k" to not delete the original file after processing
	case ".zst":
		compressProgram = "zstdmt" // zstdmt is equivalent to zstd -T0 (attempt to detect and use the number of physical CPU cores)
	case ".bz2":
		compressProgram = "bzip2"
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
		targetFilePath,                          // the .tar.zst file
		"-C",                                    // -C is used to not include the parent directory
		"/vackup-volume",                        // the directory where the files to compress are
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
	err = backend.StartContainersByName(ctxReq, cli, stoppedContainers)
	if err != nil {
		return err
	}

	return ctx.String(http.StatusCreated, "")
}
