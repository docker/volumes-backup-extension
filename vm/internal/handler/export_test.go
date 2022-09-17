package handler

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"github.com/docker/volumes-backup-extension/internal/log"
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

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	volumetypes "github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/labstack/echo"
	"github.com/stretchr/testify/require"
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
	h := New(c.Request().Context(), func() (*client.Client, error) { return setupDockerClient(t), nil })

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

	dst := filepath.Join(tmpDir, "export-destination")
	defer func() {
		// the folder that is exported from the volume.tar.gz
		if err = os.RemoveAll(dst); err != nil {
			t.Fatal(err)
		}
	}()

	if err := extractTarGz(t, dst, r); err != nil {
		t.Fatal(err)
	}

	exportedFiles := []string{"50x.html", "index.html"}

	actual := make(map[string][]byte, len(exportedFiles))
	for _, f := range exportedFiles {
		actual[f] = readFile(t, dst, f)
	}
	require.Len(t, actual, 2)

	golden := make(map[string][]byte, len(exportedFiles))
	dir := filepath.Join("testdata", "export", "vackup-volume")
	for _, f := range exportedFiles {
		golden[f] = readFile(t, dir, f+".golden")
		require.Equal(t, string(actual[f]), string(golden[f]))
	}
}

func extractTarGz(t *testing.T, dst string, gzipStream io.Reader) error {
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

		// the target location where the dir/file should be created
		target := filepath.Join(dst, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.Mkdir(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			outFile, err := os.Create(target)
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

var table = struct {
	input map[string][]string
}{
	input: map[string][]string{
		//"localhost:5000/felipecruz/10mb": {".tar.gz", ".tar.zst"},
		//"localhost:5000/felipecruz/1gb":  {".tar.gz", ".tar.zst"},
		"docker.io/felipecruz/postgres_pgdata_4gb": {".tar.gz", ".tar.zst", ".tar.bz2"},
	},
}

// go test -timeout 0 -bench=. -count 3 -run=^# | tee old.txt
// benchstat old.txt
func BenchmarkExportVolume(b *testing.B) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatal(err)
	}

	volume := "2f91f352f0ba381893b9e15ea87db0e28a88aa6e28070c07892681d7a0d6ba6b"

	for image, v := range table.input {

		setupVolume(context.Background(), cli, volume, image)

		for _, compression := range v {
			b.Run(fmt.Sprintf("compression_%s_%s", image, compression), func(b *testing.B) {
				for n := 0; n < b.N; n++ {
					export(cli, volume, compression)
				}
			})
		}

		_ = cli.VolumeRemove(context.Background(), volume, true)
	}
}

func export(cli *client.Client, volume, compression string) {
	tmpDir := os.TempDir()
	archiveFile := filepath.Join(tmpDir, volume+compression)
	defer func() {
		_ = os.Remove(archiveFile)
	}()

	// Setup
	e := echo.New()
	q := make(url.Values)
	q.Set("path", tmpDir)
	q.Set("fileName", volume+compression)
	req := httptest.NewRequest(http.MethodGet, "/?"+q.Encode(), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/volumes/:volume/export")
	c.SetParamNames("volume")
	c.SetParamValues(volume)
	h := New(c.Request().Context(), func() (*client.Client, error) { return cli, nil })

	// Export volume
	err := h.ExportVolume(c)
	if err != nil {
		log.Fatal(err)
	}
}

// setupVolume creates a volume and fills it with data from an image.
func setupVolume(ctx context.Context, cli *client.Client, volume, image string) {
	var containerID string

	defer func() {
		_ = cli.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{
			Force: true,
		})
	}()

	// Create volume
	_, err := cli.VolumeCreate(ctx, volumetypes.VolumeCreateBody{
		Driver: "local",
		Name:   volume,
	})
	if err != nil {
		log.Fatal(err)
	}

	reader, err := cli.ImagePull(ctx, image, types.ImagePullOptions{
		Platform: "linux/" + runtime.GOARCH,
	})
	if err != nil {
		log.Fatal(err)
	}

	_, err = io.Copy(os.Stdout, reader)
	if err != nil {
		log.Fatal(err)
	}

	// Populate volume
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: image,
	}, &container.HostConfig{
		Binds: []string{
			volume + ":" + "/volume-data:ro",
		},
	}, nil, nil, "")
	if err != nil {
		log.Fatal(err)
	}

	// TODO: use AutoRemove: true instead of defer
	containerID = resp.ID
}
