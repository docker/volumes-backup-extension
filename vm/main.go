package main

import (
	"bytes"
	"context"
	"flag"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/labstack/echo"
	"github.com/sirupsen/logrus"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var cli *client.Client

func init() {
	ctx := context.Background()

	var err error
	cli, err = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatal(err)
	}

	reader, err := cli.ImagePull(ctx, "docker.io/library/alpine", types.ImagePullOptions{})
	if err != nil {
		logrus.Error(err)
	}
	_, err = io.Copy(os.Stdout, reader)
	if err != nil {
		logrus.Error(err)
	}

	reader, err = cli.ImagePull(ctx, "docker.io/library/busybox", types.ImagePullOptions{})
	if err != nil {
		logrus.Error(err)
	}
	_, err = io.Copy(os.Stdout, reader)
	if err != nil {
		logrus.Error(err)
	}

	reader, err = cli.ImagePull(ctx, "docker.io/justincormack/nsenter1", types.ImagePullOptions{})
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

	startURL := ""

	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Fatal(err)
	}
	router.Listener = ln

	router.GET("/hello", hello)
	router.GET("/volumes", volumes)
	router.GET("/volumes/:volume/size", volumeSize)

	log.Fatal(router.Start(startURL))
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

func volumes(ctx echo.Context) error {
	start := time.Now()

	v, err := cli.VolumeList(ctx.Request().Context(), filters.NewArgs())
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
	resp, err := cli.VolumeInspect(ctx, volumeName)
	if err != nil {
		logrus.Error(err)
	}

	return resp.Driver
}

func calcContainers(ctx context.Context, volumeName string) []string {
	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{
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
	resp, err := cli.ContainerCreate(ctx, &container.Config{
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

	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		logrus.Error(err)
	}

	statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			panic(err)
		}
	case <-statusCh:
	}

	out, err := cli.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{ShowStdout: true})
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
		//logrus.Infof("line: %q, length: %d", s, len(s))

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

	err = cli.ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{})
	if err != nil {
		logrus.Error(err)
	}

	return m
}

func volumeSize(ctx echo.Context) error {
	start := time.Now()

	volumeName := ctx.Param("volume")
	m := calcVolSize(context.Background(), volumeName) // TODO: use request context

	logrus.Infof("/volumeSize took %s", time.Since(start))
	return ctx.JSON(http.StatusOK, m[volumeName])
}
