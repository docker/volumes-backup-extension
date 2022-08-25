package backend

import (
	"bytes"
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/felipecruz91/vackup-docker-extension/internal/log"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

type VolumeSize struct {
	Bytes int64
	Human string
}

func GetVolumesSize(ctx context.Context, cli *client.Client, volumeName string) map[string]VolumeSize {
	// Ensure the image is present before creating the container
	reader, err := cli.ImagePull(ctx, "docker.io/justincormack/nsenter1", types.ImagePullOptions{
		Platform: "linux/" + runtime.GOARCH,
	})
	if err != nil {
		log.Error(err)
	}
	_, err = io.Copy(os.Stdout, reader)
	if err != nil {
		log.Error(err)
	}

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Tty:   true,
		Cmd:   []string{"/bin/sh", "-c", "du -d 0 /var/lib/docker/volumes/" + volumeName},
		Image: "docker.io/justincormack/nsenter1",
		Labels: map[string]string{
			"com.docker.desktop.extension":        "true",
			"com.docker.desktop.extension.name":   "Volumes Backup & Share",
			"com.docker.compose.project":          "docker_volumes-backup-extension-desktop-extension",
			"com.volumes-backup-extension.action": "get-volumes-size",
			"com.volumes-backup-extension.volume": volumeName,
		},
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
	m := make(map[string]VolumeSize)
	for _, line := range lines {
		s := strings.Split(line, "\t") // e.g. 924	/var/lib/docker/volumes/my-volume
		if len(s) != 2 {
			log.Warnf("skipping line: %s", line)
			continue
		}

		size := s[0]
		path := strings.TrimSuffix(s[1], "\r")

		sizeKB, err := strconv.ParseInt(size, 10, 64)
		if err != nil {
			log.Warn(err)
			continue
		}

		if path == "/var/lib/docker/volumes/backingFsBlockDev" || path == "/var/lib/docker/volumes/metadata.db" {
			// ignore "backingFsBlockDev" and "metadata.db" system volumes
			continue
		}

		if sizeKB == 8 {
			// Apparently, inside the VM if a directory size is 8.0K, it is in fact "empty".
			// Therefore, we set it to "0B" to indicate that the directory is empty.
			sizeKB = 0
		}

		m[filepath.Base(path)] = VolumeSize{
			Bytes: sizeKB * 1000,
			Human: byteCountSI(sizeKB * 1000),
		}
	}

	err = cli.ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{})
	if err != nil {
		log.Error(err)
	}

	return m
}

// byteCountSI converts a size in bytes to a human-readable string in SI (decimal) format.
//
// e.g. 999 -> "999 B"
//
// e.g. 1000 -> "1.0 kB"
//
// e.g. 987,654,321	 -> "987.7 MB"
func byteCountSI(b int64) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB",
		float64(b)/float64(div), "kMGTPE"[exp])
}
