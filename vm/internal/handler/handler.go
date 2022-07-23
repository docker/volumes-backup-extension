package handler

import (
	"context"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"io"
	"os"
)

type Handler struct {
	DockerClient *client.Client
}

func New(ctx context.Context, cli *client.Client) *Handler {
	pullImagesIfNotPresent(ctx, cli)

	return &Handler{
		DockerClient: cli,
	}
}

func pullImagesIfNotPresent(ctx context.Context, cli *client.Client) {
	g, ctx := errgroup.WithContext(ctx)

	images := []string{
		"docker.io/library/alpine",
		"docker.io/library/busybox",
		"docker.io/justincormack/nsenter1",
	}

	for _, image := range images {
		image := image // https://golang.org/doc/faq#closures_and_goroutines
		g.Go(func() error {
			_, _, err := cli.ImageInspectWithRaw(ctx, image)
			if err != nil {
				logrus.Info("Pulling Image:", image)
				reader, err := cli.ImagePull(ctx, image, types.ImagePullOptions{})
				if err != nil {
					return err
				}
				_, err = io.Copy(os.Stdout, reader)
			}

			return nil
		})
	}

	// wait for all the pull operations to complete
	if err := g.Wait(); err == nil {
		logrus.Info("Successfully pulled all the images")
	}
}
