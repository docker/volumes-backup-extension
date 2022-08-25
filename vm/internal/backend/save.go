package backend

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/felipecruz91/vackup-docker-extension/internal/log"
	"os"
)

func Save(ctx context.Context, client *client.Client, volumeName, image string) error {
	resp, err := client.ContainerCreate(ctx, &container.Config{
		Image:        "docker.io/library/busybox",
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          []string{"/bin/sh", "-c", "cp -Rp -v /mount-volume/. /volume-data/;"},
		Labels: map[string]string{
			"com.docker.desktop.extension":      "true",
			"com.docker.desktop.extension.name": "Volumes Backup & Share",
			"com.docker.compose.project":        "docker_volumes-backup-extension-desktop-extension",
			//"com.volumes-backup-extension.trigger-ui-refresh": "true",
			"com.volumes-backup-extension.action": "save",
			"com.volumes-backup-extension.image":  image,
			"com.volumes-backup-extension.volume": volumeName,
		},
	}, &container.HostConfig{
		Binds: []string{
			volumeName + ":" + "/mount-volume",
		},
	}, nil, nil, "")
	if err != nil {
		return err
	}

	if err := client.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return err
	}

	var exitCode int64
	statusCh, errCh := client.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return err
		}
	case status := <-statusCh:
		log.Infof("status: %#+v\n", status)
		exitCode = status.StatusCode
	}

	out, err := client.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{ShowStdout: true})
	if err != nil {
		return err
	}

	_, err = stdcopy.StdCopy(os.Stdout, os.Stderr, out)
	if err != nil {
		return err
	}

	if exitCode != 0 {
		return fmt.Errorf("container exited with status code %d\n", exitCode)
	}

	_, err = client.ContainerCommit(ctx, resp.ID, types.ContainerCommitOptions{
		Reference: image,
	})

	err = client.ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{})
	if err != nil {
		return err
	}
	return err
}
