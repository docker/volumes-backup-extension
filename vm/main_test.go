package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	volumetypes "github.com/docker/docker/api/types/volume"
	"github.com/labstack/echo"
	"github.com/stretchr/testify/require"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func Test_volumes(t *testing.T) {
	volume := "7115890bd1cdf80f4cc0b8aaa9f5300281e80b4bf68170a6eb20e174774f0089"

	defer func() {
		_ = h.cli.VolumeRemove(context.Background(), volume, true)
	}()

	// Setup
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/volumes")

	// Create volume
	_, err := h.cli.VolumeCreate(c.Request().Context(), volumetypes.VolumeCreateBody{
		Driver: "local",
		Name:   volume,
	})
	require.NoError(t, err)

	// List volumes
	err = h.volumes(c)
	require.NoError(t, err)

	t.Log(rec.Body.String())
	m := map[string]VolumeData{}
	err = json.Unmarshal(rec.Body.Bytes(), &m)
	require.NoError(t, err)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, m, volume)
	require.Equal(t, "local", m[volume].Driver)
	require.Equal(t, "0B", m[volume].Size)
	require.Len(t, m[volume].Containers, 0)
}

func Test_volumeSize(t *testing.T) {
	var containerID string
	volume := "f1c149694ab1318377505d40c2431b4387a53d8d28fa814d4584e12b1ed63cfc"

	defer func() {
		_ = h.cli.ContainerRemove(context.Background(), containerID, types.ContainerRemoveOptions{
			Force: true,
		})
		_ = h.cli.VolumeRemove(context.Background(), volume, true)
	}()

	// Setup
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/volumes/:volume/size")
	c.SetParamNames("volume")
	c.SetParamValues(volume)

	// Create volume
	_, err := h.cli.VolumeCreate(c.Request().Context(), volumetypes.VolumeCreateBody{
		Driver: "local",
		Name:   volume,
	})
	require.NoError(t, err)

	reader, err := h.cli.ImagePull(c.Request().Context(), "docker.io/library/nginx:1.21", types.ImagePullOptions{})
	_, err = io.Copy(os.Stdout, reader)
	require.NoError(t, err)

	// Populate volume
	resp, err := h.cli.ContainerCreate(c.Request().Context(), &container.Config{
		Image: "docker.io/library/nginx:1.21",
	}, &container.HostConfig{
		Binds: []string{
			volume + ":" + "/usr/share/nginx/html:ro",
		},
	}, nil, nil, "")
	require.NoError(t, err)

	containerID = resp.ID

	// Get volume size
	err = h.volumeSize(c)
	require.NoError(t, err)

	size := rec.Body.String()
	t.Log(size)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Regexp(t, "\".*K\"\n", size)
}

func Test_exportHandler(t *testing.T) {
	var containerID string
	volume := "2f91f352f0ba381893b9e15ea87db0e28a88aa6e28070c07892681d7a0d6ba6b"

	defer func() {
		_ = h.cli.ContainerRemove(context.Background(), containerID, types.ContainerRemoveOptions{
			Force: true,
		})
		_ = h.cli.VolumeRemove(context.Background(), volume, true)

		exportedTarGz := filepath.Join(os.TempDir(), volume+".tar.gz")
		t.Logf("removing %s", exportedTarGz)
		if err := os.Remove(exportedTarGz); err != nil {
			t.Fatal(err)
		}
	}()

	// Setup
	e := echo.New()
	q := make(url.Values)
	q.Set("path", os.TempDir())
	q.Set("fileName", volume+".tar.gz")
	req := httptest.NewRequest(http.MethodGet, "/?"+q.Encode(), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/volumes/:volume/export")
	c.SetParamNames("volume")
	c.SetParamValues(volume)

	// Create volume
	_, err := h.cli.VolumeCreate(c.Request().Context(), volumetypes.VolumeCreateBody{
		Driver: "local",
		Name:   volume,
	})
	require.NoError(t, err)

	reader, err := h.cli.ImagePull(c.Request().Context(), "docker.io/library/nginx:1.21", types.ImagePullOptions{})
	_, err = io.Copy(os.Stdout, reader)
	require.NoError(t, err)

	// Populate volume
	resp, err := h.cli.ContainerCreate(c.Request().Context(), &container.Config{
		Image: "docker.io/library/nginx:1.21",
	}, &container.HostConfig{
		Binds: []string{
			volume + ":" + "/usr/share/nginx/html:ro",
		},
	}, nil, nil, "")
	require.NoError(t, err)

	containerID = resp.ID

	// Get volume size
	err = h.exportHandler(c)
	require.NoError(t, err)

	output := rec.Body.String()
	t.Log(output)

	require.Equal(t, http.StatusOK, rec.Code)

	// Check content of exportedFiles is correct
	r, err := os.Open(filepath.Join(os.TempDir(), volume+".tar.gz"))
	if err != nil {
		t.Fatal(err)
	}
	if err := extractTarGz(t, r); err != nil {
		t.Fatal(err)
	}
	defer func() {
		// the folder that is exported from the volume.tar.gz
		if err = os.RemoveAll("vackup-volume"); err != nil {
			t.Fatal(err)
		}
	}()

	exportedFiles := []string{"50x.html", "index.html"}

	actual := make(map[string][]byte, len(exportedFiles))
	for _, f := range exportedFiles {
		actual[f] = readFile(t, "vackup-volume", f)
	}
	require.Len(t, actual, 2)

	golden := make(map[string][]byte, len(exportedFiles))
	dir := filepath.Join("testdata", "export", "vackup-volume")
	for _, f := range exportedFiles {
		golden[f] = readFile(t, dir, f+".golden")
		require.Equal(t, string(actual[f]), string(golden[f]))
	}
}

func extractTarGz(t *testing.T, gzipStream io.Reader) error {
	t.Helper()

	uncompressedStream, err := gzip.NewReader(gzipStream)
	if err != nil {
		return err
	}

	tarReader := tar.NewReader(uncompressedStream)

	for true {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			return err
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.Mkdir(header.Name, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			outFile, err := os.Create(header.Name)
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tarReader); err != nil {
				return err
			}
			outFile.Close()

		default:
			return fmt.Errorf(
				"ExtractTarGz: uknown type: %x in %s",
				header.Typeflag,
				header.Name)
		}

	}

	return nil
}

func readFile(t *testing.T, dir string, identifier string) []byte {
	t.Helper()

	path := filepath.Join(dir, identifier)

	b, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatalf("Error opening file %s: %s", path, err)
	}

	return b
}
