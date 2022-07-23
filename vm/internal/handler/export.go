package handler

import (
	"bytes"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/felipecruz91/vackup-docker-extension/internal/backend"
	"github.com/felipecruz91/vackup-docker-extension/internal/log"
	"github.com/labstack/echo"
	"golang.org/x/sync/errgroup"
	"net/http"
	"path/filepath"
	"strings"
	"time"
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

	// Get container(s) for volume
	containerNames := backend.GetContainersForVolume(ctx.Request().Context(), h.DockerClient, volumeName)

	// Stop container(s)
	g, gCtx := errgroup.WithContext(ctx.Request().Context())

	var stoppedContainersByExtension []string
	var timeout = 10 * time.Second
	for _, containerName := range containerNames {
		containerName := containerName
		g.Go(func() error {
			// if the container linked to this volume is running then it must be stopped to ensure data integrity
			containers, err := h.DockerClient.ContainerList(ctx.Request().Context(), types.ContainerListOptions{
				Filters: filters.NewArgs(filters.Arg("name", containerName)),
			})
			if err != nil {
				return err
			}

			if len(containers) != 1 {
				log.Infof("container %s is not running, no need to stop it", containerName)
				return nil
			}

			log.Infof("stopping container %s...", containerName)
			err = h.DockerClient.ContainerStop(gCtx, containerName, &timeout)
			if err != nil {
				return err
			}

			log.Infof("container %s stopped", containerName)
			stoppedContainersByExtension = append(stoppedContainersByExtension, containerName)
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		log.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// in case it is a Windows path, replace double backslashes with a single forward slash
	path = strings.Replace(path, "\\\\", "/", -1)

	// add a leading slash before the drive letter and remove the extra colon after the drive letter
	path = "/" + strings.Replace(path, ":", "", 1)

	// TODO: quote path in case it includes spaces
	log.Infof("path cleaned up in case it is a Windows path: %s", path)

	// Export

	// fmt.Sprintf("tar -zcvf /vackup/%s /vackup-volume", fileName)
	cmd := []string{
		"tar",
		"-zcvf",
		filepath.Join("/vackup", filepath.Base(fileName)),
		"/vackup-volume",
	}
	cmdJoined := strings.Join(cmd, " ")
	log.Infof("cmdJoined: %s", cmdJoined)

	resp, err := h.DockerClient.ContainerCreate(ctx.Request().Context(), &container.Config{
		Image:        "docker.io/library/busybox",
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          []string{"/bin/sh", "-c", cmdJoined},
	}, &container.HostConfig{
		Binds: []string{
			volumeName + ":" + "/vackup-volume",
			path + ":" + "/vackup",
		},
	}, nil, nil, "")
	if err != nil {
		log.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	if err := h.DockerClient.ContainerStart(ctx.Request().Context(), resp.ID, types.ContainerStartOptions{}); err != nil {
		log.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err)
	}

	statusCh, errCh := h.DockerClient.ContainerWait(ctx.Request().Context(), resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			log.Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
	case <-statusCh:
	}

	out, err := h.DockerClient.ContainerLogs(ctx.Request().Context(), resp.ID, types.ContainerLogsOptions{ShowStdout: true})
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

	// TODO: check if container exited with error code
	// if so, return internal server error!

	err = h.DockerClient.ContainerRemove(ctx.Request().Context(), resp.ID, types.ContainerRemoveOptions{})
	if err != nil {
		log.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Start container(s)
	g, gCtx = errgroup.WithContext(ctx.Request().Context())
	for _, containerName := range stoppedContainersByExtension {
		containerName := containerName
		g.Go(func() error {
			log.Infof("starting container %s...", containerName)
			err := h.DockerClient.ContainerStart(gCtx, containerName, types.ContainerStartOptions{})
			if err != nil {
				return err
			}

			log.Infof("container %s started", containerName)
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		log.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return ctx.String(http.StatusOK, "")
}
