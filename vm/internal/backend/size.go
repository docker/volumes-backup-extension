package backend

import (
	"bytes"
	"context"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/felipecruz91/vackup-docker-extension/internal/log"
	"path/filepath"
	"strings"
)

func GetVolumesSize(ctx context.Context, cli *client.Client, volumeName string) map[string]string {
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Tty:   true,
		Cmd:   []string{"/bin/sh", "-c", "du -d 0 -h /var/lib/docker/volumes/" + volumeName},
		Image: "docker.io/justincormack/nsenter1",
	}, &container.HostConfig{
		PidMode:    "host",
		Privileged: true,
	}, nil, nil, "")
	if err != nil {
		log.Error(err)
	}

	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		log.Error(err)
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
		log.Error(err)
	}

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(out)
	if err != nil {
		log.Error(err)
	}

	output := buf.String()

	lines := strings.Split(strings.TrimSuffix(output, "\n"), "\n")
	m := make(map[string]string)
	for _, line := range lines {
		s := strings.Split(line, "\t") // e.g. 41.5M	/var/lib/docker/volumes/my-volume
		if len(s) != 2 {
			log.Warnf("skipping line: %s", line)
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

	err = cli.ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{})
	if err != nil {
		log.Error(err)
	}

	return m
}
