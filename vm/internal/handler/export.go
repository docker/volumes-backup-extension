package handler

import (
	"bytes"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/felipecruz91/vackup-docker-extension/internal/backend"
	"github.com/felipecruz91/vackup-docker-extension/internal/log"
	"github.com/labstack/echo"
	"net/http"
	"path/filepath"
	"strings"
)

func (h *Handler) ExportVolume(ctx echo.Context) error {
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

	// fix windows path
	if strings.Contains(path, ":") {
		path = strings.Replace(path, "C:\\", "/c/", 1)
		path = strings.Replace(path, "\\", "/", -1)
		path = "/mnt/host" + path // TODO: Only if running WSL2
	}

	//log.Infof("%s, %s", runtime.GOOS, runtime.GOARCH) // because it's running inside a container, it returns linux, amd64

	//cmd := exec.Command("wsl", "-l", "-v")
	//out, err := cmd.CombinedOutput()
	//if err != nil {
	//	log.Fatalf("cmd.Run() failed with %s\n", err)
	//}
	//fmt.Printf("combined out:\n%s\n", string(out))

	log.Infof("path replaced: %s", path)

	// Stop container(s)
	stoppedContainers, err := backend.StopContainersAttachedToVolume(ctx.Request().Context(), h.DockerClient, volumeName)
	if err != nil {
		log.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// in case it is a Windows path:
	//if strings.Contains(path, ":") {
	//	// add a leading slash before the drive letter and remove the extra colon after the drive letter
	//	path = "/" + strings.Replace(path, ":", "", 1)
	//	log.Infof("added leading slash to path and removed colon char: %s", path)
	//
	//	// replace double backslashes with a single forward slash
	//	path = strings.ReplaceAll(path, "\\", "/")
	//	log.Infof("replaced double backslahes with a single forward slash: %s", path)
	//
	//	// TODO: quote path in case it includes spaces
	//	//log.Infof("path cleaned up in case it is a Windows path: %s", path)
	//}

	// Export
	//fmt.Sprintf("tar -zcvf /vackup/%s /vackup-volume", fileName)
	cmd := []string{
		"tar",
		"-zcvf",
		"/vackup" + "/" + filepath.Base(fileName),
		"/vackup-volume",
	}

	cmdJoined := strings.Join(cmd, " ")
	log.Infof("cmdJoined: %s", cmdJoined)

	binds := []string{
		volumeName + ":" + "/vackup-volume",
		path + ":" + "/vackup",
	}
	log.Infof("binds: %+v", binds)

	resp, err := h.DockerClient.ContainerCreate(ctx.Request().Context(), &container.Config{
		Image:        "docker.io/library/busybox",
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          []string{"/bin/sh", "-c", cmdJoined},
		User:         "root",
	}, &container.HostConfig{
		Binds: binds,
	}, nil, nil, "")
	if err != nil {
		log.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	if err := h.DockerClient.ContainerStart(ctx.Request().Context(), resp.ID, types.ContainerStartOptions{}); err != nil {
		log.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err)
	}

	// TODO: check if container exited with error code in other handlers
	// if so, return internal server error!
	var exitCode int64
	statusCh, errCh := h.DockerClient.ContainerWait(ctx.Request().Context(), resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			log.Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
	case status := <-statusCh:
		log.Infof("status: %#+v\n", status)
		exitCode = status.StatusCode
	}

	out, err := h.DockerClient.ContainerLogs(ctx.Request().Context(), resp.ID, types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true})
	if err != nil {
		log.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(out)
	if err != nil {
		log.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	output := buf.String()

	log.Info(output)

	if exitCode != 0 {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Errorf("container exited with status code %d, output: %s\n", exitCode, output))
	}

	// TODO: FIX THE FOLLOWING ISSUE ON WINDOWS:
	//C:\Users\felipe>docker logs 02bbad243e50
	//tar: can't open '/vackup/exported.tar.gz': Operation not permitted

	err = h.DockerClient.ContainerRemove(ctx.Request().Context(), resp.ID, types.ContainerRemoveOptions{})
	if err != nil {
		log.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Start container(s)
	err = backend.StartContainersAttachedToVolume(ctx.Request().Context(), h.DockerClient, stoppedContainers)
	if err != nil {
		log.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return ctx.String(http.StatusOK, "")
}
