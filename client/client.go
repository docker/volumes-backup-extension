//go:build !windows
// +build !windows

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/sirupsen/logrus"
)

type VolumePushOptions struct {
	RegistryAuth string
}

type VolumePullOptions struct {
	RegistryAuth string
}

type Client interface {
	Push(ctx context.Context, reference string, volume string, options VolumePushOptions) error
	Pull(ctx context.Context, reference string, volume string, options VolumePullOptions) error
}

type cl struct {
	httpc http.Client
}

type ClientOpt func(Client) error

// New returns a new volume client
func New(opts ...ClientOpt) (Client, error) {
	// TODO: pass extension name when calling the ./volumes-share-client or use an env variable
	// The socket name in the **host** is not the one defined in the "metadata.json" of the extension.
	// It is the extension installation directory name followed by ".sock".
	extensionInstallationDir := "felipecruz_vackup-docker-extension"
	hostSocketName := extensionInstallationDir + ".sock"
	c := &cl{
		httpc: http.Client{
			Transport: &http.Transport{
				DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
					hd, err := os.UserHomeDir()
					if err != nil {
						return nil, err
					}
					var socket string
					switch runtime.GOOS {
					case "darwin":
						// e.g. "/Users/felipecruz/.docker/ext-sockets/felipecruz_vackup-docker-extension.sock"
						socket = filepath.Join(hd, ".docker", "ext-sockets", hostSocketName)
					case "linux":
						// e.g. "/home/felipecruz/.docker/desktop/ext-sockets/felipecruz_vackup-docker-extension.sock"
						socket = filepath.Join(hd, ".docker", "desktop", "ext-sockets", hostSocketName)
					}
					logrus.Infof("unix socket: %s", socket)
					return net.Dial("unix", socket)
				},
			},
		},
	}

	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
	}

	return c, nil
}

type PushRequest struct {
	Reference string `json:"reference"`
}

func (c *cl) Push(ctx context.Context, ref string, volume string, options VolumePushOptions) error {
	auth := options.RegistryAuth
	request := PushRequest{
		Reference: ref,
	}

	data, err := json.Marshal(request)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("http://unix/volumes/%s/push", volume), bytes.NewBuffer(data))
	if err != nil {
		return err
	}

	req.Header.Set("X-Registry-Auth", auth)
	req.Header.Set("Content-Type", "application/json")

	res, err := c.httpc.Do(req)
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(res.Body)
		return errors.New(string(b))
	}

	return err
}

type PullRequest struct {
	Reference string `json:"reference"`
}

func (c *cl) Pull(ctx context.Context, reference string, volume string, options VolumePullOptions) error {
	auth := options.RegistryAuth

	request := PullRequest{
		Reference: reference,
	}

	data, err := json.Marshal(request)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("http://unix/volumes/%s/pull", volume), bytes.NewBuffer(data))
	if err != nil {
		return err
	}

	req.Header.Set("X-Registry-Auth", auth)
	req.Header.Set("Content-Type", "application/json")

	res, err := c.httpc.Do(req)
	if res.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(res.Body)
		return errors.New(string(b))
	}

	return err
}
