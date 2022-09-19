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
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatal(err)
	}

	volume := "2f91f352f0ba381893b9e15ea87db0e28a88aa6e28070c07892681d7a0d6ba6b"
	image := "docker.io/library/nginx:1.21"
	mountPath := "/usr/share/nginx/html:ro"

	setupVolume(context.Background(), cli, volume, image, mountPath)

	tmpDir := os.TempDir()

	compressions := []string{".tar.gz"} // TODO: add ".tar.zst" and ".tar.bz2"

	for _, compression := range compressions {
		t.Run(fmt.Sprintf("TestExportVolume_%s_%s", image, compression), func(t *testing.T) {
			archiveFile := filepath.Join(tmpDir, volume+compression)

			defer func() {
				_ = os.Remove(archiveFile)
			}()

			// Export volume
			rec := export(cli, volume, tmpDir, compression)

			require.NoError(t, err)
			require.Equal(t, http.StatusCreated, rec.Code)

			// Check content of exportedFiles is correct
			r, err := os.Open(filepath.Join(tmpDir, volume+compression))
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

		})
	}

	_ = cli.VolumeRemove(context.Background(), volume, true)
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
		"docker.io/felipecruz/postgres_pgdata_4gb": {".tar.gz", ".tar.zst"},
	},
}

func BenchmarkExportVolume(b *testing.B) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatal(err)
	}

	volume := "2f91f352f0ba381893b9e15ea87db0e28a88aa6e28070c07892681d7a0d6ba6b"
	tmpDir := os.TempDir()

	for image, v := range table.input {
		setupVolume(context.Background(), cli, volume, image, "/volume-data:ro")

		for _, compression := range v {
			archiveFileName := filepath.Join(tmpDir, volume+compression)
			b.Run(fmt.Sprintf("compression_%s_%s", image, compression), func(b *testing.B) {
				for n := 0; n < b.N; n++ {
					export(cli, volume, tmpDir, compression)
					archiveFile, _ := os.Stat(archiveFileName)
					b.Logf("%s - Size: %d bytes.", archiveFile.Name(), archiveFile.Size())
				}
			})
		}

		_ = cli.VolumeRemove(context.Background(), volume, true)
	}
}

func export(cli *client.Client, volume, path, compression string) *httptest.ResponseRecorder {

	// Setup
	e := echo.New()
	q := make(url.Values)
	q.Set("path", path)
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
	return rec
}

// setupVolume creates a volume and fills it with data from an image.
func setupVolume(ctx context.Context, cli *client.Client, volume, image, mountPath string) {
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
			volume + ":" + mountPath,
		},
	}, nil, nil, "")
	if err != nil {
		log.Fatal(err)
	}

	_ = cli.ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{
		Force: true,
	})
}
