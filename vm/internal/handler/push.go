package handler

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"github.com/containerd/containerd/remotes"
	"github.com/containerd/containerd/remotes/docker"
	"github.com/docker/distribution/reference"
	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/volumes-backup-extension/internal/backend"
	"github.com/docker/volumes-backup-extension/internal/log"
	"github.com/labstack/echo"
	"github.com/sirupsen/logrus"
	"net/http"
	"strings"
)

type PushRequest struct {
	Reference         string `json:"reference"`
	Base64EncodedAuth string `json:"base64EncodedAuth"`
}

type PushErrorLine struct {
	ErrorDetail ErrorDetail `json:"errorDetail"`
	Error       string      `json:"error"`
}
type ErrorDetail struct {
	Message string `json:"message"`
}

// PushVolume pushes a volume to a registry.
// The user must be previously authenticated to the registry with `docker login <registry>`, otherwise it returns 401 StatusUnauthorized.
func (h *Handler) PushVolume(ctx echo.Context) error {
	ctxReq := ctx.Request().Context()

	var request PushRequest
	if err := ctx.Bind(&request); err != nil {
		return err
	}

	volumeName := ctx.Param("volume")
	log.Infof("volumeName: %s", volumeName)
	log.Infof("reference: %s", request.Reference)
	log.Infof("received push request for volume %s\n", volumeName)

	cli, err := h.DockerClient()
	if err != nil {
		return err
	}
	defer func() {
		h.ProgressCache.Lock()
		delete(h.ProgressCache.m, volumeName)
		h.ProgressCache.Unlock()
		_ = backend.TriggerUIRefresh(ctxReq, cli)
	}()

	h.ProgressCache.Lock()
	h.ProgressCache.m[volumeName] = "push"
	h.ProgressCache.Unlock()

	err = backend.TriggerUIRefresh(ctxReq, cli)
	if err != nil {
		return err
	}

	// To provide backwards compatibility with older versions of Docker Desktop,
	// we're passing the encoded auth in the body of the request instead of in the headers.
	// encodedAuth := ctx.Request().Header.Get("X-Registry-Auth")
	if request.Base64EncodedAuth == "" {
		request.Base64EncodedAuth = "Cg==" // from running: echo "" | base64
	}

	if volumeName == "" {
		return ctx.String(http.StatusBadRequest, "volume is required")
	}

	parsedRef, err := reference.ParseAnyReference(request.Reference)
	if err != nil {
		return ctx.String(http.StatusBadRequest, err.Error())
	}
	log.Infof("parsedRef.String(): %s", parsedRef.String())

	// Stop container(s)
	stoppedContainers, err := backend.StopContainersAttachedToVolume(ctxReq, cli, volumeName)
	if err != nil {
		return err
	}

	// Push volume as an OCI artifact to registry
	authConfig := &dockertypes.AuthConfig{}
	authJSON := base64.NewDecoder(base64.URLEncoding, strings.NewReader(request.Base64EncodedAuth))
	if err := json.NewDecoder(authJSON).Decode(authConfig); err != nil {
		logrus.Infof("failed to decode auth config: %s\n", err)
		return err
	}

	resolver := newDockerResolver(authConfig.Username, authConfig.Password, true)
	if err := backend.Push(ctxReq, request.Reference, volumeName, resolver); err != nil {
		logrus.Infof("failed to push volume %s to ref %s: %s\n", request.Reference, request.Reference, err)
		return err
	}

	// Start container(s)
	err = backend.StartContainersAttachedToVolume(ctxReq, cli, stoppedContainers)
	if err != nil {
		return err
	}

	return ctx.String(http.StatusCreated, "")
}

func newDockerResolver(username, password string, insecure bool) remotes.Resolver {
	client := http.DefaultClient
	if insecure {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
	}
	opts := docker.ResolverOptions{
		Hosts: func(s string) ([]docker.RegistryHost, error) {
			return []docker.RegistryHost{
				{
					Authorizer: docker.NewDockerAuthorizer(
						docker.WithAuthCreds(func(s string) (string, string, error) {
							return username, password, nil
						})),
					Host:         "docker.io",
					Scheme:       "https",
					Path:         "/v2",
					Capabilities: docker.HostCapabilityPull | docker.HostCapabilityResolve | docker.HostCapabilityPush,
					Client:       client,
				},
			}, nil
		},
	}

	return docker.NewResolver(opts)
}
