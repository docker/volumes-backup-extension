package handler

import (
	"archive/tar"
	"compress/gzip"
	"context"
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
	"runtime"
	"strings"
	"testing"
)

func TestExportVolume(t *testing.T) {
	var containerID string
	volume := "2f91f352f0ba381893b9e15ea87db0e28a88aa6e28070c07892681d7a0d6ba6b"
	cli := setupDockerClient(t)
	tmpDir := os.TempDir()

	defer func() {
		_ = cli.ContainerRemove(context.Background(), containerID, types.ContainerRemoveOptions{
			Force: true,
		})
		_ = cli.VolumeRemove(context.Background(), volume, true)

		exportedTarGz := filepath.Join(tmpDir, volume+".tar.gz")
		t.Logf("removing %s", exportedTarGz)
		if err := os.Remove(exportedTarGz); err != nil {
			t.Log(err)
		}
	}()

	// Setup
	e := echo.New()
	q := make(url.Values)
	q.Set("path", tmpDir)
	q.Set("fileName", volume+".tar.gz")
	req := httptest.NewRequest(http.MethodGet, "/?"+q.Encode(), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/volumes/:volume/export")
	c.SetParamNames("volume")
	c.SetParamValues(volume)
	h := New(c.Request().Context(), setupDockerClient(t))

	// Create volume
	_, err := cli.VolumeCreate(c.Request().Context(), volumetypes.VolumeCreateBody{
		Driver: "local",
		Name:   volume,
	})
	if err != nil {
		t.Fatal(err)
	}

	reader, err := cli.ImagePull(c.Request().Context(), "docker.io/library/nginx:1.21", types.ImagePullOptions{
		Platform: "linux/" + runtime.GOARCH,
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = io.Copy(os.Stdout, reader)
	if err != nil {
		t.Fatal(err)
	}

	// Populate volume
	resp, err := cli.ContainerCreate(c.Request().Context(), &container.Config{
		Image: "docker.io/library/nginx:1.21",
	}, &container.HostConfig{
		Binds: []string{
			volume + ":" + "/usr/share/nginx/html:ro",
		},
	}, nil, nil, "")
	if err != nil {
		t.Fatal(err)
	}

	containerID = resp.ID

	// Export volume
	err = h.ExportVolume(c)
	require.NoError(t, err)

	require.Equal(t, http.StatusCreated, rec.Code)

	// Check content of exportedFiles is correct
	r, err := os.Open(filepath.Join(tmpDir, volume+".tar.gz"))
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

	if runtime.GOOS == "windows" {
		// replace CRLF (\r\n) with LF (\n)
		output := strings.Replace(string(b), "\r\n", "\n", -1)
		b = []byte(output)
	}

	return b
}
