package backend

import (
	"context"
	"github.com/bugsnag/bugsnag-go/v2"
	"github.com/docker/docker/client"
	"github.com/docker/volumes-backup-extension/internal/log"
)

func GetVolumeDriver(ctx context.Context, cli *client.Client, volumeName string) string {
	resp, err := cli.VolumeInspect(ctx, volumeName)
	if err != nil {
		log.Error(err)
		_ = bugsnag.Notify(err, ctx)
	}

	return resp.Driver
}
