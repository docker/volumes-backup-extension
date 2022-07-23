package backend

import (
	"context"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
	"strings"
)

func GetContainersForVolume(ctx context.Context, cli *client.Client, volumeName string) []string {
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

	return containerNames
}
