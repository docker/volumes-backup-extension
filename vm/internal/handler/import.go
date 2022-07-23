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
	"time"
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

	filePathDir := filepath.Dir(path)
	fileName := filepath.Base(path)

	log.Infof("volumeName: %s", volumeName)
	log.Infof("path: %s", path)
	log.Infof("filePathDir: %s", filePathDir)
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

	// Import
	resp, err := h.DockerClient.ContainerCreate(ctx.Request().Context(), &container.Config{
		Image:        "docker.io/library/busybox",
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          []string{"/bin/sh", "-c", "tar -xvzf /vackup"},
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
