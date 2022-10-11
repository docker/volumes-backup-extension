package backend

import (
	"context"
	"io"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/bugsnag/bugsnag-go/v2"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/docker/volumes-backup-extension/internal"
	"github.com/docker/volumes-backup-extension/internal/log"
	"golang.org/x/sync/errgroup"
)

func GetContainersForVolume(ctx context.Context, cli *client.Client, volumeName string, specialFilters filters.Args) []string {
	// add our filters filterArgs
	specialFilters.Add("volume", volumeName)

	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{
		All:     true,
		Filters: specialFilters,
	})
	if err != nil {
		log.Error(err)
		_ = bugsnag.Notify(err, ctx)
	}

	containerNames := make([]string, 0, len(containers))
	for _, c := range containers {
		containerNames = append(containerNames, strings.TrimPrefix(c.Names[0], "/"))
	}

	return containerNames
}

func StopRunningContainersAttachedToVolume(ctx context.Context, cli *client.Client, volumeName string) ([]string, error) {
	var stoppedContainersByExtension []string
	var timeout = 10 * time.Second

	containerNames := GetContainersForVolume(ctx, cli, volumeName, filters.NewArgs(filters.Arg("status", "running")))

	g, gCtx := errgroup.WithContext(ctx)
	for _, containerName := range containerNames {
		containerName := containerName
		g.Go(func() error {
			// if the container linked to this volume is running then it must be stopped to ensure data integrity
			containers, err := cli.ContainerList(gCtx, types.ContainerListOptions{
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
			err = cli.ContainerStop(gCtx, containerName, &timeout)
			if err != nil {
				return err
			}

			log.Infof("container %s stopped", containerName)
			stoppedContainersByExtension = append(stoppedContainersByExtension, containerName)
			return nil
		})
	}

	return containerNames, g.Wait()
}

func StartContainersByName(ctx context.Context, cli *client.Client, containers []string) error {
	g, gCtx := errgroup.WithContext(ctx)

	for _, containerName := range containers {
		containerName := containerName
		g.Go(func() error {
			log.Infof("starting container %s...", containerName)
			err := cli.ContainerStart(gCtx, containerName, types.ContainerStartOptions{})
			if err != nil {
				return err
			}

			log.Infof("container %s started", containerName)
			return nil
		})
	}

	return g.Wait()
}

// TriggerUIRefresh starts a container to notify the extension UI to reload the progress actions from the cache.
// The container uses the label "com.volumes-backup-extension.trigger-ui-refresh=true" for that purpose and is auto-removed when exited.
// The extension UI is listening for container events with that label. Once an event is received, the extension UI makes a request to the `/progress` endpoint
// to load the actions in progress from the cache.
func TriggerUIRefresh(ctx context.Context, cli *client.Client) error {
	// Ensure the image is present before creating the container
	if _, _, err := cli.ImageInspectWithRaw(ctx, internal.BusyboxImage); err != nil {
		reader, err := cli.ImagePull(ctx, internal.BusyboxImage, types.ImagePullOptions{
			Platform: "linux/" + runtime.GOARCH,
		})
		if err != nil {
			return err
		}
		_, err = io.Copy(os.Stdout, reader)
		if err != nil {
			return err
		}
	}

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image:        internal.BusyboxImage,
		AttachStdout: true,
		AttachStderr: true,
		Labels: map[string]string{
			"com.docker.desktop.extension":                    "true",
			"com.docker.desktop.extension.name":               "Volumes Backup & Share",
			"com.docker.compose.project":                      "docker_volumes-backup-extension-desktop-extension",
			"com.volumes-backup-extension.trigger-ui-refresh": "true",
		},
	}, &container.HostConfig{
		AutoRemove: true,
	}, nil, nil, "")
	if err != nil {
		return err
	}

	return cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{})
}
