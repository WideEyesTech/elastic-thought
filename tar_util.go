package elasticthought

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/couchbaselabs/logg"
)

type tarFile struct {
	Name string
	Body string
}

// Opens tar.gz stream
func openTarGzStream(url string) (*tar.Reader, error) {

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	gzipReader, err := gzip.NewReader(resp.Body)
	if err != nil {
		return nil, err
	}
	tarReader := tar.NewReader(gzipReader)

	return tarReader, nil

}

func untarWithToc(reader io.Reader, destDirectory string) ([]string, error) {

	toc := []string{}
	tr := tar.NewReader(reader)

	// Iterate through the files in the archive.
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			// end of tar archive
			break
		}
		if err != nil {
			return nil, err
		}

		if err := writeToDest(hdr, tr, destDirectory); err != nil {
			return nil, err
		}

		// add to toc
		toc = append(toc, hdr.Name)

	}

	return toc, nil

}

// Given a reader, wrap in a tar.gz reader and write all entries
// to destDirectory.  Also return a table of contents.
func untarGzWithToc(reader io.Reader, destDirectory string) ([]string, error) {

	gzipReader, err := gzip.NewReader(reader)
	if err != nil {
		return nil, err
	}
	return untarWithToc(gzipReader, destDirectory)
}

func writeToDest(hdr *tar.Header, tr *tar.Reader, destDirectory string) error {

	destPath := filepath.Join(destDirectory, hdr.Name)

	if strings.HasPrefix(hdr.Name, ".") {
		logg.LogTo("TRAINING_JOB", "ignore hidden file: %v", destPath)
		return nil
	}

	logg.LogTo("TRAINING_JOB", "writeToDest called with: %v hdr: %+v", destPath, hdr)

	switch hdr.Typeflag {
	case tar.TypeDir:
		logg.LogTo("TRAINING_JOB", "%v is a directory", destPath)
		// does dir exist? if not, make it
		logg.LogTo("TRAINING_JOB", "calling mkdir on %v", destPath)
		if err := mkdir(destPath); err != nil {
			logg.LogTo("TRAINING_JOB", "mkdir failed on %v", destPath)
			return err
		}

	default:

		/*
			// make the directory in case it doesn't already exist
			destPathDir := path.Dir(destPath)
			if err := mkdir(destPathDir); err != nil {
				logg.LogTo("TRAINING_JOB", "mkdir failed on %v", destPath)
				return err
			}
		*/

		logg.LogTo("TRAINING_JOB", "calling os.Create on %v", destPath)
		f, err := os.Create(destPath)
		if err != nil {
			logg.LogTo("TRAINING_JOB", "calling os.Create failed on %v", destPath)
			return err
		}
		w := bufio.NewWriter(f)
		defer w.Flush()
		_, err = io.Copy(w, tr)
		if err != nil {
			logg.LogTo("TRAINING_JOB", "io.Copy failed: %v", err)
			return err
		}

	}
	return nil

}

func createArchive(buf *bytes.Buffer, tarFiles []tarFile) {

	// Create a new tar archive.
	tw := tar.NewWriter(buf)

	for _, file := range tarFiles {
		hdr := &tar.Header{
			Name: file.Name,
			Size: int64(len(file.Body)),
			Uid:  100,
			Gid:  101,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			log.Fatalln(err)
		}
		if _, err := tw.Write([]byte(file.Body)); err != nil {
			log.Fatalln(err)
		}
	}
	// Make sure to check the error on Close.
	if err := tw.Close(); err != nil {
		log.Fatalln(err)
	}

}

// TempDir returns the default directory to use for temporary files.
func TempDir() string {

	dir := os.Getenv("TMPDIR")
	if dir == "" {
		dir = "/tmp"
	}
	return dir
}