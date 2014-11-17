package elasticthought

import (
	"archive/tar"
	"bytes"
	"io"
	"log"
	"strings"
	"testing"

	"github.com/couchbaselabs/go.assert"
	"github.com/couchbaselabs/logg"
)

func TestTransform(t *testing.T) {

	dataset := Dataset{
		TrainingDataset: TrainingDataset{
			SplitPercentage: 0.5,
		},
		TestDataset: TestDataset{
			SplitPercentage: 0.5,
		},
	}

	splitter := DatasetSplitter{
		Dataset: dataset,
	}

	// Create a test tar archive
	buf := new(bytes.Buffer)
	createTestArchive(buf)

	// Open the tar archive for reading.
	r := bytes.NewReader(buf.Bytes())
	tr := tar.NewReader(r)

	// create two writers
	bufTrain := new(bytes.Buffer)
	bufTest := new(bytes.Buffer)
	twTrain := tar.NewWriter(bufTrain)
	twTest := tar.NewWriter(bufTest)

	// pass these into transform
	err := splitter.transform(tr, twTrain, twTest)
	assert.True(t, err == nil)

	// assert that the both the training and test tar archives
	// have been split 50/50 over each "label" folder, so each "label"
	// folder should now have one example each.  Ie, one "foo/x.txt" and
	// one "bar/x.txt".  Also, the resulting split sets must be disjoint.
	buffers := []*bytes.Buffer{bufTrain, bufTest}
	for _, buffer := range buffers {
		readerVerify := bytes.NewReader(buffer.Bytes())
		trVerify := tar.NewReader(readerVerify)

		seenFoos := map[string]string{}
		seenBars := map[string]string{}
		numFoo := 0
		numBar := 0
		for {
			hdr, err := trVerify.Next()
			if err == io.EOF {
				// end of tar archive
				break
			}
			if err != nil {
				log.Fatalln(err)
			}
			if strings.HasPrefix(hdr.Name, "foo") {
				numFoo += 1
				// make sure it's the first time seeing this filename
				if _, ok := seenFoos[hdr.Name]; ok {
					logg.LogPanic("Not first time seeing: %v", hdr.Name)
				}
				seenFoos[hdr.Name] = hdr.Name
			}
			if strings.HasPrefix(hdr.Name, "bar") {
				numBar += 1
				if _, ok := seenBars[hdr.Name]; ok {
					logg.LogPanic("Not first time seeing: %v", hdr.Name)
				}
				seenBars[hdr.Name] = hdr.Name

			}

		}

		assert.Equals(t, numFoo, 1)
		assert.Equals(t, numBar, 1)

	}

}

func createTestArchive(buf *bytes.Buffer) {

	// Create a new tar archive.
	tw := tar.NewWriter(buf)

	var files = []struct {
		Name, Body string
	}{
		{"foo/1.txt", "Hello 1."},
		{"foo/2.txt", "Hello 2."},
		{"bar/1.txt", "Hello bar 1."},
		{"bar/2.txt", "Hello bar 2."},
	}
	for _, file := range files {
		hdr := &tar.Header{
			Name: file.Name,
			Size: int64(len(file.Body)),
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