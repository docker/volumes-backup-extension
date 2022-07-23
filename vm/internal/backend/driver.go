package backend

import (
	"context"
	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
)

func GetVolumeDriver(ctx context.Context, cli *client.Client, volumeName string) string {
	resp, err := cli.VolumeInspect(ctx, volumeName)
	if err != nil {
		logrus.Error(err)
	}

	return resp.Driver
}
