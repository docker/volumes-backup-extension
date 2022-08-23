package main

import (
	"context"
	"fmt"
	"os"

	"github.com/docker/distribution/reference"
	"github.com/docker/docker/registry"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "Docker Credentials client",
		Usage: "Read the Docker credentials.",
		Commands: []*cli.Command{
			{
				Name:        "get-creds",
				UsageText:   "docker-credentials-client get-creds REFERENCE",
				Description: "Returns the Docker credentials (in base64) for the registry that is specified in the REFERENCE.",
				Action: func(c *cli.Context) error {
					ref := c.Args().Get(0)

					parsedRef, err := reference.ParseNormalizedNamed(ref)
					if err != nil {
						return err
					}

					repoInfo, err := registry.ParseRepositoryInfo(parsedRef)
					if err != nil {
						return err
					}

					authConfig, err := resolveAuthConfig(context.Background(), repoInfo.Index)
					if err != nil {
						return err
					}

					encodedAuth, err := encodeAuthToBase64(authConfig)
					if err != nil {
						return err
					}

					fmt.Print(encodedAuth)
					return nil
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		logrus.Fatal(err)
	}
}
