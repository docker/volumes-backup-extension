package handler

import (
	"context"
	"github.com/bugsnag/bugsnag-go/v2"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/felipecruz91/vackup-docker-extension/internal"
	"github.com/felipecruz91/vackup-docker-extension/internal/log"
	"golang.org/x/sync/errgroup"
	"io"
	"os"
	"runtime"
)

type Handler struct {
	DockerClient  *client.Client
	ProgressCache *ProgressCache
}

func New(ctx context.Context, cli *client.Client) *Handler {
	pullImagesIfNotPresent(ctx, cli)

	return &Handler{
		DockerClient: cli,
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
					_ = bugsnag.Notify(err)
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
