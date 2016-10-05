package tar

import (
	"archive/tar"
	"io"
	"os"
	"path"

	"github.com/pkg/errors"
)

type extractor func(r *tar.Reader, h *tar.Header, root string) error

// maps from tar.Header.TypeFlag to a function that knows how to extract it.
var extractorMap = map[byte]extractor{
	tar.TypeReg:           extractFile,
	tar.TypeRegA:          extractFile,
	tar.TypeDir:           extractDir,
	tar.TypeXGlobalHeader: ignore, // AUR packages fill this with the commit id.
}

// ExtractAll extracts the tar file in r and puts it into root.
// Currently only supports files and directories.
func ExtractAll(reader io.Reader, root string) error {
	r := tar.NewReader(reader)
	for {
		header, err := r.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return errors.Wrap(err, "r.Next()")
		}
		extractFunc, ok := extractorMap[header.Typeflag]
		if !ok {
			return errors.Errorf("Unknown TypeFlag %x for %s", header.Typeflag, header.Name)
		}
		err = extractFunc(r, header, root)
		if err != nil {
			return err
		}
	}
}

func ignore(r *tar.Reader, h *tar.Header, root string) error {
	return nil
}

func extractFile(r *tar.Reader, h *tar.Header, root string) error {
	dest := path.Join(root, h.Name)
	fileInfo := h.FileInfo()
	f, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, fileInfo.Mode())
	if err != nil {
		return err
	}
	defer func() {
		closeErr := f.Close()
		if err == nil {
			err = closeErr
		}
	}()

	_, err = io.Copy(f, r)
	return err
}

func extractDir(r *tar.Reader, h *tar.Header, root string) error {
	dest := path.Join(root, h.Name)
	fileInfo := h.FileInfo()
	return os.MkdirAll(dest, fileInfo.Mode())
}
