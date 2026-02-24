package archive

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/argoproj/argo-workflows/v4/errors"
	"github.com/argoproj/argo-workflows/v4/util"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

type flusher interface {
	Flush() error
}

// TarGzToWriter tar.gz's the source path to the supplied writer
func TarGzToWriter(ctx context.Context, sourcePath string, level int, w io.Writer) error {
	sourcePath, err := filepath.Abs(sourcePath)
	if err != nil {
		return errors.InternalErrorf("getting absolute path: %v", err)
	}
	logger := logging.RequireLoggerFromContext(ctx)
	logger.WithField("source", sourcePath).Info(ctx, "tarring")
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
	gzw, err := gzip.NewWriterLevel(w, level)
	if err != nil {
		return errors.InternalWrapError(err)
	}
	defer util.Close(gzw)
	tw := tar.NewWriter(gzw)
	defer util.Close(tw)

	if sourceFi.IsDir() {
		return tarDir(ctx, sourcePath, tw)
	}
	return tarFile(sourcePath, tw)
}

// ZipToWriter zip the source path to the supplied writer
func ZipToWriter(ctx context.Context, sourcePath string, zw *zip.Writer) error {
	sourcePath, err := filepath.Abs(sourcePath)
	if err != nil {
		return errors.InternalErrorf("getting absolute path: %v", err)
	}
	logger := logging.RequireLoggerFromContext(ctx)
	logger.WithField("source", sourcePath).Info(ctx, "zipping")
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

	if sourceFi.IsDir() {
		return zipDir(ctx, sourcePath, zw)
	}
	return zipFile(sourcePath, zw)
}

func tarDir(ctx context.Context, sourcePath string, tw *tar.Writer) error {
	baseName := filepath.Base(sourcePath)
	count := 0
	logger := logging.RequireLoggerFromContext(ctx).WithField("sourcePath", sourcePath)
	err := filepath.Walk(sourcePath, func(fpath string, info os.FileInfo, err error) error {
		if err != nil {
			return errors.InternalWrapError(err)
		}
		// build the name to be used in the archive
		nameInArchive, err := filepath.Rel(sourcePath, fpath)
		if err != nil {
			return errors.InternalWrapError(err)
		}
		nameInArchive = filepath.ToSlash(filepath.Join(baseName, nameInArchive))
		logger.WithField("nameInArchive", nameInArchive).Debug(ctx, "writing")
		count++

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
		f, err := os.Open(filepath.Clean(fpath))
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
	logger.WithFields(logging.Fields{
		"count":  count,
		"source": sourcePath,
	}).Info(ctx, "archived files/dirs")
	return err
}

func tarFile(sourcePath string, tw *tar.Writer) error {
	f, err := os.Open(filepath.Clean(sourcePath))
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
	return err
}

func zipDir(ctx context.Context, sourcePath string, zw *zip.Writer) error {
	baseName := filepath.Base(sourcePath)
	count := 0
	logger := logging.RequireLoggerFromContext(ctx)
	err := filepath.Walk(sourcePath, func(fpath string, info os.FileInfo, err error) error {
		if err != nil {
			return errors.InternalWrapError(err)
		}
		if info.IsDir() {
			return nil
		}
		// build the name to be used in the archive
		nameInArchive, err := filepath.Rel(sourcePath, fpath)
		if err != nil {
			return errors.InternalWrapError(err)
		}
		nameInArchive = filepath.Join(baseName, nameInArchive)
		logger.WithField("name", nameInArchive).Info(ctx, "writing")
		count++

		fileWriter, err := zw.Create(nameInArchive)
		if err != nil {
			return errors.InternalWrapError(err)
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		f, err := os.Open(filepath.Clean(fpath))
		if err != nil {
			return errors.InternalWrapError(err)
		}
		defer f.Close()

		// copy file data into zip writer
		_, err = io.Copy(fileWriter, f)
		if err != nil {
			return err
		}

		return nil
	})
	logger.WithFields(logging.Fields{
		"count":  count,
		"source": sourcePath,
	}).Info(ctx, "archive[zip] files/dirs")
	return err
}

func zipFile(sourcePath string, zw *zip.Writer) error {
	f, err := os.Open(filepath.Clean(sourcePath))
	if err != nil {
		return errors.InternalWrapError(err)
	}
	defer util.Close(f)
	fileWriter, err := zw.Create(sourcePath)
	if err != nil {
		return errors.InternalWrapError(err)
	}
	_, err = io.Copy(fileWriter, f)
	return err
}
