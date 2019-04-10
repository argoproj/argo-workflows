package archive

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"

	"github.com/argoproj/argo/errors"
	"github.com/argoproj/argo/util"
	log "github.com/sirupsen/logrus"
)

type flusher interface {
	Flush() error
}

// TarGzToWriter tar.gz's the source path to the supplied writer
func TarGzToWriter(sourcePath string, w io.Writer) error {
	sourcePath, err := filepath.Abs(sourcePath)
	if err != nil {
		return errors.InternalErrorf("getting absolute path: %v", err)
	}
	log.Infof("Taring %s", sourcePath)
	sourceFi, err := os.Stat(sourcePath)
	if err != nil {
		if os.IsNotExist(err) {
			return errors.New(errors.CodeNotFound, err.Error())
		}
		return errors.InternalWrapError(err)
	}
	if !sourceFi.Mode().IsRegular() && !sourceFi.IsDir() {
		return errors.InternalErrorf("%s is not a regular file or directory", sourcePath)
	}
	if flush, ok := w.(flusher); ok {
		defer func() { _ = flush.Flush() }()
	}
	gzw := gzip.NewWriter(w)
	defer util.Close(gzw)
	tw := tar.NewWriter(gzw)
	defer util.Close(tw)

	if sourceFi.IsDir() {
		return tarDir(sourcePath, tw)
	}
	return tarFile(sourcePath, tw)
}

func tarDir(sourcePath string, tw *tar.Writer) error {
	baseName := filepath.Base(sourcePath)
	return filepath.Walk(sourcePath, func(fpath string, info os.FileInfo, err error) error {
		if err != nil {
			return errors.InternalWrapError(err)
		}
		// build the name to be used in the archive
		nameInArchive, err := filepath.Rel(sourcePath, fpath)
		if err != nil {
			return errors.InternalWrapError(err)
		}
		nameInArchive = filepath.Join(baseName, nameInArchive)
		log.Infof("writing %s", nameInArchive)

		var header *tar.Header
		if (info.Mode() & os.ModeSymlink) != 0 {
			linkTarget, err := os.Readlink(fpath)
			if err != nil {
				return errors.InternalWrapError(err)
			}
			header, err = tar.FileInfoHeader(info, filepath.ToSlash(linkTarget))
			if err != nil {
				return errors.InternalWrapError(err)
			}
		} else {
			header, err = tar.FileInfoHeader(info, info.Name())
			if err != nil {
				return errors.InternalWrapError(err)
			}
		}
		header.Name = nameInArchive

		err = tw.WriteHeader(header)
		if err != nil {
			return errors.InternalWrapError(err)
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		f, err := os.Open(fpath)
		if err != nil {
			return errors.InternalWrapError(err)
		}

		// copy file data into tar writer
		_, err = io.Copy(tw, f)
		closeErr := f.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
		return nil
	})
}

func tarFile(sourcePath string, tw *tar.Writer) error {
	f, err := os.Open(sourcePath)
	if err != nil {
		return errors.InternalWrapError(err)
	}
	defer util.Close(f)
	info, err := f.Stat()
	if err != nil {
		return errors.InternalWrapError(err)
	}
	header, err := tar.FileInfoHeader(info, f.Name())
	if err != nil {
		return errors.InternalWrapError(err)
	}
	header.Name = filepath.Base(sourcePath)
	err = tw.WriteHeader(header)
	if err != nil {
		return errors.InternalWrapError(err)
	}
	_, err = io.Copy(tw, f)
	if err != nil {
		return err
	}
	return nil
}
