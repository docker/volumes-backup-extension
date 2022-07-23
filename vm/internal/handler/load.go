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
	"time"
)

func (h *Handler) LoadImage(ctx echo.Context) error {
	volumeName := ctx.Param("volume")
	image := ctx.QueryParam("image")

	if volumeName == "" {
		return ctx.String(http.StatusBadRequest, "volume is required")
	}
	if image == "" {
		return ctx.String(http.StatusBadRequest, "image is required")
	}

	log.Infof("volumeName: %s", volumeName)
	log.Infof("image: %s", image)

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

	// Load
	resp, err := h.DockerClient.ContainerCreate(ctx.Request().Context(), &container.Config{
		Image:        image,
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          []string{"/bin/sh", "-c", "cp -Rp /volume-data/. /mount-volume/;"},
	}, &container.HostConfig{
		Binds: []string{
			volumeName + ":" + "/mount-volume",
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
