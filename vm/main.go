package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/labstack/echo"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var h handler

func init() {
	ctx := context.Background()

	var err error
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatal(err)
	}

	h = handler{cli: cli}

	reader, err := h.cli.ImagePull(ctx, "docker.io/library/alpine", types.ImagePullOptions{})
	if err != nil {
		logrus.Error(err)
	}
	_, err = io.Copy(os.Stdout, reader)
	if err != nil {
		logrus.Error(err)
	}

	reader, err = h.cli.ImagePull(ctx, "docker.io/library/busybox", types.ImagePullOptions{})
	if err != nil {
		logrus.Error(err)
	}
	_, err = io.Copy(os.Stdout, reader)
	if err != nil {
		logrus.Error(err)
	}

	reader, err = h.cli.ImagePull(ctx, "docker.io/justincormack/nsenter1", types.ImagePullOptions{})
	if err != nil {
		logrus.Error(err)
	}
	_, err = io.Copy(os.Stdout, reader)
	if err != nil {
		logrus.Error(err)
	}
}

func main() {
	var socketPath string
	flag.StringVar(&socketPath, "socket", "/run/guest/extension-vackup.sock", "Unix domain socket to listen on")
	flag.Parse()

	_ = os.RemoveAll(socketPath)

	logrus.New().Infof("Starting listening on %s\n", socketPath)
	router := echo.New()
	router.HideBanner = true

	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Fatal(err)
	}
	router.Listener = ln

	router.GET("/hello", hello)
	router.GET("/volumes", h.volumes)
	router.GET("/volumes/:volume/size", h.volumeSize)
	router.GET("/volumes/:volume/export", h.exportHandler)
	router.GET("/volumes/:volume/import", importHandler)
	router.GET("/volumes/:volume/save", saveHandler)
	router.GET("/volumes/:volume/load", loadHandler)

	// Start server
	go func() {
		if err := router.Start(""); err != nil && err != http.ErrServerClosed {
			logrus.Fatal("shutting down the server")
		}
	}()

	// Wait for interrupt signal to gracefully shut down the server with a timeout of 10 seconds.
	// Use a buffered channel to avoid missing signals as recommended for signal.Notify
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := router.Shutdown(ctx); err != nil {
		logrus.Fatal(err)
	}
}

func hello(ctx echo.Context) error {
	return ctx.String(http.StatusOK, "hello")
}

type VolumesResponse struct {
	sync.RWMutex
	data map[string]VolumeData
}

type VolumeData struct {
	Driver     string
	Size       string
	Containers []string
}

type handler struct {
	cli *client.Client
}

func (h *handler) volumes(ctx echo.Context) error {
	start := time.Now()

	v, err := h.cli.VolumeList(ctx.Request().Context(), filters.NewArgs())
	if err != nil {
		logrus.Error(err)
	}

	var res = VolumesResponse{
		data: map[string]VolumeData{},
	}

	var wg sync.WaitGroup
	// Calculating the volume size by spinning a container that execs "du " **per volume** is too time-consuming.
	// To reduce the time it takes, we get the volumes size by running only one container that execs "du"
	// into the /var/lib/docker/volumes inside the VM.
	volumesSize := calcVolSize(ctx.Request().Context(), "*")
	res.Lock()
	for k, v := range volumesSize {
		entry, ok := res.data[k]
		if !ok {
			res.data[k] = VolumeData{
				Size: v,
			}
			continue
		}
		entry.Size = v
		res.data[k] = entry
	}
	res.Unlock()

	for _, vol := range v.Volumes {
		wg.Add(2)
		go func(volumeName string) {
			defer wg.Done()
			driver := calcVolDriver(context.Background(), volumeName) // TODO: use request context
			res.Lock()
			defer res.Unlock()
			entry, ok := res.data[volumeName]
			if !ok {
				res.data[volumeName] = VolumeData{
					Driver: driver,
				}
				return
			}
			entry.Driver = driver
			res.data[volumeName] = entry
		}(vol.Name)

		go func(volumeName string) {
			defer wg.Done()
			containers := calcContainers(context.Background(), volumeName) // TODO: use request context
			res.Lock()
			defer res.Unlock()
			entry, ok := res.data[volumeName]
			if !ok {
				res.data[volumeName] = VolumeData{
					Containers: containers,
				}
				return
			}
			entry.Containers = containers
			res.data[volumeName] = entry
		}(vol.Name)
	}

	wg.Wait()
	logrus.Infof("/volumes took %s", time.Since(start))
	return ctx.JSON(http.StatusOK, res.data)
}

func calcVolDriver(ctx context.Context, volumeName string) string {
	resp, err := h.cli.VolumeInspect(ctx, volumeName)
	if err != nil {
		logrus.Error(err)
	}

	return resp.Driver
}

func calcContainers(ctx context.Context, volumeName string) []string {
	containers, err := h.cli.ContainerList(ctx, types.ContainerListOptions{
		All:     true,
		Filters: filters.NewArgs(filters.Arg("volume", volumeName)),
	})
	if err != nil {
		logrus.Error(err)
	}

	containerNames := make([]string, 0, len(containers))
	for _, c := range containers {
		containerNames = append(containerNames, strings.TrimPrefix(c.Names[0], "/"))
	}

	logrus.Info(containerNames)

	return containerNames
}

func calcVolSize(ctx context.Context, volumeName string) map[string]string {
	resp, err := h.cli.ContainerCreate(ctx, &container.Config{
		Tty:   true,
		Cmd:   []string{"/bin/sh", "-c", "du -d 0 -h /var/lib/docker/volumes/" + volumeName},
		Image: "docker.io/justincormack/nsenter1",
	}, &container.HostConfig{
		PidMode:    "host",
		Privileged: true,
	}, nil, nil, "")
	if err != nil {
		logrus.Error(err)
	}

	if err := h.cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		logrus.Error(err)
	}

	statusCh, errCh := h.cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			panic(err)
		}
	case <-statusCh:
	}

	out, err := h.cli.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{ShowStdout: true})
	if err != nil {
		logrus.Error(err)
	}

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(out)
	if err != nil {
		logrus.Error(err)
	}

	output := buf.String()

	lines := strings.Split(strings.TrimSuffix(output, "\n"), "\n")
	m := make(map[string]string)
	for _, line := range lines {
		s := strings.Split(line, "\t") // e.g. 41.5M	/var/lib/docker/volumes/my-volume
		if len(s) != 2 {
			logrus.Warnf("skipping line: %s", line)
			continue
		}

		size := s[0]
		path := strings.TrimSuffix(s[1], "\r")

		if path == "/var/lib/docker/volumes/backingFsBlockDev" || path == "/var/lib/docker/volumes/metadata.db" {
			// ignore "backingFsBlockDev" and "metadata.db" system volumes
			continue
		}

		if size == "8.0K" {
			// Apparently, inside the VM if a directory size is 8.0K, it is in fact "empty".
			// Therefore, we set it to "0B" to indicate that the directory is empty.
			size = "0B"
		}

		m[filepath.Base(path)] = size
	}

	//logrus.Info(m)

	err = h.cli.ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{})
	if err != nil {
		logrus.Error(err)
	}

	return m
}

func (h *handler) volumeSize(ctx echo.Context) error {
	start := time.Now()

	volumeName := ctx.Param("volume")
	m := calcVolSize(ctx.Request().Context(), volumeName)

	logrus.Infof("/volumeSize took %s", time.Since(start))
	return ctx.JSON(http.StatusOK, m[volumeName])
}

func (h *handler) exportHandler(ctx echo.Context) error {
	start := time.Now()

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

	logrus.Infof("volumeName: %s", volumeName)
	logrus.Infof("path: %s", path)
	logrus.Infof("fileName: %s", fileName)

	// Get container(s) for volume
	containerNames := calcContainers(ctx.Request().Context(), volumeName)

	// Stop container(s)
	g, gCtx := errgroup.WithContext(ctx.Request().Context())

	var stoppedContainersByExtension []string
	var timeout = 10 * time.Second
	for _, containerName := range containerNames {
		containerName := containerName
		g.Go(func() error {
			// if the container linked to this volume is running then it must be stopped to ensure data integrity
			containers, err := h.cli.ContainerList(ctx.Request().Context(), types.ContainerListOptions{
				Filters: filters.NewArgs(filters.Arg("name", containerName)),
			})
			if err != nil {
				return err
			}

			if len(containers) != 1 {
				logrus.Infof("container %s is not running, no need to stop it", containerName)
				return nil
			}

			logrus.Infof("stopping container %s...", containerName)
			err = h.cli.ContainerStop(gCtx, containerName, &timeout)
			if err != nil {
				return err
			}

			logrus.Infof("container %s stopped", containerName)
			stoppedContainersByExtension = append(stoppedContainersByExtension, containerName)
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		logrus.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// in case it is a Windows path, replace double backslashes with a single forward slash
	path = strings.Replace(path, "\\\\", "/", -1)

	// add a leading slash before the drive letter and remove the extra colon after the drive letter
	path = "/" + strings.Replace(path, ":", "", 1)

	// TODO: quote path in case it includes spaces
	log.Printf("path cleaned up in case it is a Windows path: %s", path)

	// Export

	// fmt.Sprintf("tar -zcvf /vackup/%s /vackup-volume", fileName)
	cmd := []string{
		"tar",
		"-zcvf",
		filepath.Join("/vackup", filepath.Base(fileName)),
		"/vackup-volume",
	}
	cmdJoined := strings.Join(cmd, " ")
	logrus.Infof("cmdJoined: %s", cmdJoined)

	resp, err := h.cli.ContainerCreate(ctx.Request().Context(), &container.Config{
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
		logrus.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	if err := h.cli.ContainerStart(ctx.Request().Context(), resp.ID, types.ContainerStartOptions{}); err != nil {
		logrus.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err)
	}

	statusCh, errCh := h.cli.ContainerWait(ctx.Request().Context(), resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			logrus.Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
	case <-statusCh:
	}

	out, err := h.cli.ContainerLogs(ctx.Request().Context(), resp.ID, types.ContainerLogsOptions{ShowStdout: true})
	if err != nil {
		logrus.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(out)
	if err != nil {
		logrus.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	output := buf.String()

	logrus.Info(output)

	// TODO: check if container exited with error code
	// if so, return internal server error!

	err = h.cli.ContainerRemove(ctx.Request().Context(), resp.ID, types.ContainerRemoveOptions{})
	if err != nil {
		logrus.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Start container(s)
	g, gCtx = errgroup.WithContext(ctx.Request().Context())
	for _, containerName := range stoppedContainersByExtension {
		containerName := containerName
		g.Go(func() error {
			logrus.Infof("starting container %s...", containerName)
			err := h.cli.ContainerStart(gCtx, containerName, types.ContainerStartOptions{})
			if err != nil {
				return err
			}

			logrus.Infof("container %s started", containerName)
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		logrus.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	logrus.Infof(fmt.Sprintf("/volumes/%s/export took %s", volumeName, time.Since(start)))
	return ctx.String(http.StatusOK, "")
}

func importHandler(ctx echo.Context) error {
	start := time.Now()

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

	logrus.Infof("volumeName: %s", volumeName)
	logrus.Infof("path: %s", path)
	logrus.Infof("filePathDir: %s", filePathDir)
	logrus.Infof("fileName: %s", fileName)

	// Get container(s) for volume
	containerNames := calcContainers(ctx.Request().Context(), volumeName)

	// Stop container(s)
	g, gCtx := errgroup.WithContext(ctx.Request().Context())

	var stoppedContainersByExtension []string
	var timeout = 10 * time.Second
	for _, containerName := range containerNames {
		containerName := containerName
		g.Go(func() error {
			// if the container linked to this volume is running then it must be stopped to ensure data integrity
			containers, err := h.cli.ContainerList(ctx.Request().Context(), types.ContainerListOptions{
				Filters: filters.NewArgs(filters.Arg("name", containerName)),
			})
			if err != nil {
				return err
			}

			if len(containers) != 1 {
				logrus.Infof("container %s is not running, no need to stop it", containerName)
				return nil
			}

			logrus.Infof("stopping container %s...", containerName)
			err = h.cli.ContainerStop(gCtx, containerName, &timeout)
			if err != nil {
				return err
			}

			logrus.Infof("container %s stopped", containerName)
			stoppedContainersByExtension = append(stoppedContainersByExtension, containerName)
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		logrus.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Import
	resp, err := h.cli.ContainerCreate(ctx.Request().Context(), &container.Config{
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
		logrus.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	if err := h.cli.ContainerStart(ctx.Request().Context(), resp.ID, types.ContainerStartOptions{}); err != nil {
		logrus.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err)
	}

	statusCh, errCh := h.cli.ContainerWait(ctx.Request().Context(), resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			logrus.Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
	case <-statusCh:
	}

	out, err := h.cli.ContainerLogs(ctx.Request().Context(), resp.ID, types.ContainerLogsOptions{ShowStdout: true})
	if err != nil {
		logrus.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(out)
	if err != nil {
		logrus.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	output := buf.String()

	logrus.Info(output)

	err = h.cli.ContainerRemove(ctx.Request().Context(), resp.ID, types.ContainerRemoveOptions{})
	if err != nil {
		logrus.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Start container(s)
	g, gCtx = errgroup.WithContext(ctx.Request().Context())
	for _, containerName := range stoppedContainersByExtension {
		containerName := containerName
		g.Go(func() error {
			logrus.Infof("starting container %s...", containerName)
			err := h.cli.ContainerStart(gCtx, containerName, types.ContainerStartOptions{})
			if err != nil {
				return err
			}

			logrus.Infof("container %s started", containerName)
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		logrus.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	logrus.Infof(fmt.Sprintf("/volumes/%s/import took %s", volumeName, time.Since(start)))
	return ctx.String(http.StatusOK, "")
}

func saveHandler(ctx echo.Context) error {
	start := time.Now()

	volumeName := ctx.Param("volume")
	image := ctx.QueryParam("image")

	if volumeName == "" {
		return ctx.String(http.StatusBadRequest, "volume is required")
	}
	if image == "" {
		return ctx.String(http.StatusBadRequest, "image is required")
	}

	logrus.Infof("volumeName: %s", volumeName)
	logrus.Infof("image: %s", image)

	// Get container(s) for volume
	containerNames := calcContainers(ctx.Request().Context(), volumeName)

	// Stop container(s)
	g, gCtx := errgroup.WithContext(ctx.Request().Context())

	var stoppedContainersByExtension []string
	var timeout = 10 * time.Second
	for _, containerName := range containerNames {
		containerName := containerName
		g.Go(func() error {
			// if the container linked to this volume is running then it must be stopped to ensure data integrity
			containers, err := h.cli.ContainerList(ctx.Request().Context(), types.ContainerListOptions{
				Filters: filters.NewArgs(filters.Arg("name", containerName)),
			})
			if err != nil {
				return err
			}

			if len(containers) != 1 {
				logrus.Infof("container %s is not running, no need to stop it", containerName)
				return nil
			}

			logrus.Infof("stopping container %s...", containerName)
			err = h.cli.ContainerStop(gCtx, containerName, &timeout)
			if err != nil {
				return err
			}

			logrus.Infof("container %s stopped", containerName)
			stoppedContainersByExtension = append(stoppedContainersByExtension, containerName)
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		logrus.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Save
	resp, err := h.cli.ContainerCreate(ctx.Request().Context(), &container.Config{
		Image:        "docker.io/library/busybox",
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          []string{"/bin/sh", "-c", "cp -Rp /mount-volume/. /volume-data/;"},
	}, &container.HostConfig{
		Binds: []string{
			volumeName + ":" + "/mount-volume",
		},
	}, nil, nil, "save-volume")
	if err != nil {
		logrus.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	if err := h.cli.ContainerStart(ctx.Request().Context(), resp.ID, types.ContainerStartOptions{}); err != nil {
		logrus.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err)
	}

	statusCh, errCh := h.cli.ContainerWait(ctx.Request().Context(), resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			logrus.Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
	case <-statusCh:
	}

	out, err := h.cli.ContainerLogs(ctx.Request().Context(), resp.ID, types.ContainerLogsOptions{ShowStdout: true})
	if err != nil {
		logrus.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(out)
	if err != nil {
		logrus.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	output := buf.String()

	logrus.Info(output)

	_, err = h.cli.ContainerCommit(ctx.Request().Context(), resp.ID, types.ContainerCommitOptions{
		Reference: image,
	})
	if err != nil {
		logrus.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	err = h.cli.ContainerRemove(ctx.Request().Context(), resp.ID, types.ContainerRemoveOptions{})
	if err != nil {
		logrus.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Start container(s)
	g, gCtx = errgroup.WithContext(ctx.Request().Context())
	for _, containerName := range stoppedContainersByExtension {
		containerName := containerName
		g.Go(func() error {
			logrus.Infof("starting container %s...", containerName)
			err := h.cli.ContainerStart(gCtx, containerName, types.ContainerStartOptions{})
			if err != nil {
				return err
			}

			logrus.Infof("container %s started", containerName)
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		logrus.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	logrus.Infof(fmt.Sprintf("/volumes/%s/save took %s", volumeName, time.Since(start)))
	return ctx.String(http.StatusOK, "")
}

func loadHandler(ctx echo.Context) error {
	start := time.Now()

	volumeName := ctx.Param("volume")
	image := ctx.QueryParam("image")

	if volumeName == "" {
		return ctx.String(http.StatusBadRequest, "volume is required")
	}
	if image == "" {
		return ctx.String(http.StatusBadRequest, "image is required")
	}

	logrus.Infof("volumeName: %s", volumeName)
	logrus.Infof("image: %s", image)

	// Get container(s) for volume
	containerNames := calcContainers(ctx.Request().Context(), volumeName)

	// Stop container(s)
	g, gCtx := errgroup.WithContext(ctx.Request().Context())

	var stoppedContainersByExtension []string
	var timeout = 10 * time.Second
	for _, containerName := range containerNames {
		containerName := containerName
		g.Go(func() error {
			// if the container linked to this volume is running then it must be stopped to ensure data integrity
			containers, err := h.cli.ContainerList(ctx.Request().Context(), types.ContainerListOptions{
				Filters: filters.NewArgs(filters.Arg("name", containerName)),
			})
			if err != nil {
				return err
			}

			if len(containers) != 1 {
				logrus.Infof("container %s is not running, no need to stop it", containerName)
				return nil
			}

			logrus.Infof("stopping container %s...", containerName)
			err = h.cli.ContainerStop(gCtx, containerName, &timeout)
			if err != nil {
				return err
			}

			logrus.Infof("container %s stopped", containerName)
			stoppedContainersByExtension = append(stoppedContainersByExtension, containerName)
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		logrus.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Load
	resp, err := h.cli.ContainerCreate(ctx.Request().Context(), &container.Config{
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
		logrus.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	if err := h.cli.ContainerStart(ctx.Request().Context(), resp.ID, types.ContainerStartOptions{}); err != nil {
		logrus.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err)
	}

	statusCh, errCh := h.cli.ContainerWait(ctx.Request().Context(), resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			logrus.Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
	case <-statusCh:
	}

	out, err := h.cli.ContainerLogs(ctx.Request().Context(), resp.ID, types.ContainerLogsOptions{ShowStdout: true})
	if err != nil {
		logrus.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(out)
	if err != nil {
		logrus.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	output := buf.String()

	logrus.Info(output)

	err = h.cli.ContainerRemove(ctx.Request().Context(), resp.ID, types.ContainerRemoveOptions{})
	if err != nil {
		logrus.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Start container(s)
	g, gCtx = errgroup.WithContext(ctx.Request().Context())
	for _, containerName := range stoppedContainersByExtension {
		containerName := containerName
		g.Go(func() error {
			logrus.Infof("starting container %s...", containerName)
			err := h.cli.ContainerStart(gCtx, containerName, types.ContainerStartOptions{})
			if err != nil {
				return err
			}

			logrus.Infof("container %s started", containerName)
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		logrus.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	logrus.Infof(fmt.Sprintf("/volumes/%s/load took %s", volumeName, time.Since(start)))
	return ctx.String(http.StatusOK, "")
}
