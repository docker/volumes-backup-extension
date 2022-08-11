package main

import (
	"context"
	"os"

	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/registry"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

func main() {
	var extensionDir string

	app := &cli.App{
		Name:  "vpush",
		Usage: "Push/Pull volumes",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "extension-dir",
				Required:    true,
				Usage:       "The directory name where the extension is installed.",
				Destination: &extensionDir,
			},
		},
		Before: func(c *cli.Context) error {
			return nil
		},
		Commands: []*cli.Command{
			{
				Name:      "pull",
				UsageText: "vs-client pull REFERENCE VOLUME",
				Action: func(c *cli.Context) error {
					ref := c.Args().Get(0)
					volume := c.Args().Get(1)

					volumeClient, err := New(extensionDir)
					if err != nil {
						return err
					}

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

					encodedAuth, err := encodeAuthToBase64(types.AuthConfig(authConfig))
					if err != nil {
						return err
					}

					return volumeClient.Pull(context.Background(), parsedRef.String(), volume, VolumePullOptions{
						RegistryAuth: encodedAuth,
					})
				},
			},
			{
				Name:      "push",
				UsageText: "vs-client push REFERENCE VOLUME",
				Action: func(c *cli.Context) error {
					ref := c.Args().Get(0)
					volume := c.Args().Get(1)

					volumeClient, err := New(extensionDir)
					if err != nil {
						return err
					}

					parsedRef, err := reference.ParseNormalizedNamed(ref)
					if err != nil {
						return err
					}

					logrus.Infof("parsedRef: %+v", parsedRef)

					repoInfo, err := registry.ParseRepositoryInfo(parsedRef)
					if err != nil {
						return err
					}

					logrus.Infof("repoInfo.Index: %+v", repoInfo.Index)

					authConfig, err := resolveAuthConfig(context.Background(), repoInfo.Index)
					if err != nil {
						return err
					}

					logrus.Infof("authConfig.Username: %s", authConfig.Username)
					logrus.Infof("authConfig.ServerAddress: %s", authConfig.ServerAddress)

					encodedAuth, err := encodeAuthToBase64(types.AuthConfig(authConfig))
					if err != nil {
						return err
					}

					return volumeClient.Push(context.Background(), parsedRef.String(), volume, VolumePushOptions{
						RegistryAuth: encodedAuth,
					})
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		logrus.Fatal(err)
	}
}
