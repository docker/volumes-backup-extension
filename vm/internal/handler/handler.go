package handler

import (
	"context"
	"io"
	"os"
	"runtime"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/volumes-backup-extension/internal"
	"github.com/docker/volumes-backup-extension/internal/log"
	"golang.org/x/sync/errgroup"
)

type Handler struct {
	DockerClient  func() (*client.Client, error)
	ProgressCache *ProgressCache
}

func New(ctx context.Context, cliFactory func() (*client.Client, error)) *Handler {
	cli, err := cliFactory()
	if err != nil {
		log.Fatal(err)
	}
	pullImagesIfNotPresent(ctx, cli)

	return &Handler{
		DockerClient: cliFactory,
		ProgressCache: &ProgressCache{
			m: make(map[string]string),
		},
	}
}

func pullImagesIfNotPresent(ctx context.Context, cli *client.Client) {
	g, ctx := errgroup.WithContext(ctx)

	images := []string{
		internal.BusyboxImage,
		internal.NsenterImage,
	}

	for _, image := range images {
		image := image // https://golang.org/doc/faq#closures_and_goroutines
		g.Go(func() error {
			_, _, err := cli.ImageInspectWithRaw(ctx, image)
			if err != nil {
				log.Info("Pulling Image:", image)
				reader, err := cli.ImagePull(ctx, image, types.ImagePullOptions{
					Platform: "linux/" + runtime.GOARCH,
				})
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
		log.Info("Successfully pulled all the images")
	}
}
