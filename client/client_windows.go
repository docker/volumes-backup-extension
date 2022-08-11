//go:build windows
// +build windows

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
	"strings"

	"github.com/sirupsen/logrus"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	npipe "gopkg.in/natefinch/npipe.v2"
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

// New returns a new volume client
func New(extensionDir string) (Client, error) {
	logrus.Infof("extensionDir not used on Windows as the socket name doesn't depend on it.")

	metadataExtensionSocket := "ext.sock" // name of the socket in metadata.json

	c := &cl{
		httpc: http.Client{
			Transport: &http.Transport{
				DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
					var socket string
					metadataExtensionSocket = strings.TrimSuffix(strings.ReplaceAll(metadataExtensionSocket, "-", ""), ".sock")
					socket = `\\.\pipe\dockerDesktopPlugin` + cases.Title(language.English, cases.NoLower).String(metadataExtensionSocket)
					logrus.Infof("npipe: %s", socket)
					return npipe.Dial(socket)
				},
			},
		},
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
	if res.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(res.Body)
		return errors.New(string(b))
	}

	return err
}
