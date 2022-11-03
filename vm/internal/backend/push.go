package backend

import (
	"context"
	"fmt"
	"github.com/containerd/containerd/log"
	"github.com/containerd/containerd/reference/docker"
	"github.com/containerd/containerd/remotes"
	"github.com/docker/distribution/reference"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"

	"oras.land/oras-go/pkg/content"
	"oras.land/oras-go/pkg/oras"

	"io/ioutil"
	"os"
	"path/filepath"
)

func Push(ctx context.Context, ref string, volume string, resolver remotes.Resolver) error {
	//dir, err := ioutil.TempDir("/tmp", volume)
	//if err != nil {
	//	return err
	//}

	//archive := filepath.Join(dir, volume)

	dataDir := filepath.Join("/var/lib/docker/volumes", volume, "_data")
	logrus.Infof("dataDir: %s", dataDir)
	//if err := dirTar(dataDir, archive); err != nil {
	//	return err
	//}
	//defer os.Remove(archive)

	return push(ctx, ref, volume, dataDir, resolver)
}

//
//func dirTar(path string, dest string) error {
//	pr, pw := io.Pipe()
//	w := gzip.NewWriter(pw)
//
//	go func() {
//		tw := tar.NewWriter(w)
//
//		_ = filepath.Walk(path, func(filePath string, f os.FileInfo, e error) error {
//			if e != nil {
//				return e
//			}
//
//			if !f.Mode().IsRegular() {
//				return nil
//			}
//
//			header, err := tar.FileInfoHeader(f, f.Name())
//			if err != nil {
//				return err
//			}
//
//			header.Name = strings.TrimPrefix(strings.Replace(filePath, path, "", -1), string(filepath.Separator))
//
//			err = tw.WriteHeader(header)
//			if err != nil {
//				return err
//			}
//
//			fi, err := os.Open(filePath)
//			if err != nil {
//				return err
//			}
//
//			if _, err := io.Copy(tw, fi); err != nil {
//				return err
//			}
//
//			return fi.Close()
//		})
//
//		w.Close()
//		pw.Close()
//		tw.Close()
//	}()
//
//	f, err := os.Create(dest)
//	if err != nil {
//		return err
//	}
//
//	defer f.Close()
//
//	_, err = io.Copy(f, pr)
//	if err != nil {
//		return err
//	}
//
//	return pr.Close()
//}

func withMutedContext(ctx context.Context) context.Context {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)
	logger.SetOutput(ioutil.Discard)
	return log.WithLogger(ctx, logrus.NewEntry(logger))
}

func push(ctx context.Context, ref string, volume string, dir string, resolver remotes.Resolver) error {
	r, err := reference.ParseNormalizedNamed(ref)
	if err != nil {
		return err
	}
	taggedReference := docker.TagNameOnly(r)

	logrus.Infof("Pushing %s to %s\n", dir, ref)

	tmpVolumeDir, err := ioutil.TempDir("/tmp", volume)
	if err != nil {
		return err
	}
	logrus.Infof("tmpVolumeDir: %s", tmpVolumeDir)

	fileStore := content.NewFileStore(tmpVolumeDir)
	defer fileStore.Close()

	f, err := os.Create(filepath.Join(tmpVolumeDir, "config.json"))
	if err != nil {
		return err
	}
	logrus.Infof("config.json path: %s", f.Name())

	_, err = f.Write([]byte("{}"))
	if err != nil {
		return err
	}
	f.Close()

	config, err := fileStore.Add("config.json", "application/vnd.docker.volume.v1+tar.gz", f.Name())
	if err != nil {
		return err
	}

	desc, err := fileStore.Add(fmt.Sprintf("%s.tar.gz", volume), "application/gzip", dir)
	if err != nil {
		return err
	}

	pushContents := []ocispec.Descriptor{desc}
	logrus.Info(pushContents)

	ctx = withMutedContext(ctx)
	desc, err = oras.Push(ctx, resolver, taggedReference.String(), fileStore, pushContents, oras.WithConfig(config))
	if err != nil {
		return err
	}

	logrus.Infof("Pushed to %s with digest %s\n", ref, desc.Digest)

	return nil
}
