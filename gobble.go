package main

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/pkg/errors"
)

var redresserStore map[string]redresser

type redresser struct {
	mountPath    string
	originalPath string
	checksum     string
}

var files = map[string]string{
	"test.txt": "test.txt",
}

func init() {
	redresserStore = make(map[string]redresser)
	for mountPath, originalPath := range files {
		r := redresser{
			mountPath:    mountPath,
			originalPath: originalPath,
			checksum:     calcChecksumPanic(originalPath),
		}
		redresserStore[mountPath] = r
	}
}

func calcChecksum(filePath string) (string, error) {
	checksum, err := md5sum(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to calculate checksum for %s during init()", filePath)
	}

	return checksum, nil
}

func calcChecksumPanic(filePath string) string {
	checksum, err := md5sum(filePath)
	if err != nil {
		panic(fmt.Errorf("failed to calculate checksum for %s during init()", filePath))
	}

	return checksum
}

func md5sum(filePath string) (string, error) {
	var checksum string

	file, err := os.Open(filePath)
	if err != nil {
		return checksum, errors.Wrapf(err, "failed to open file %s", filePath)
	}
	defer file.Close()

	hash := md5.New()
	if _, err = io.Copy(hash, file); err != nil {
		return checksum, errors.Wrap(err, "failed to calculate md5sum")
	}
	checksum = hex.EncodeToString(hash.Sum(nil))

	return checksum, nil
}

func copy(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	return out.Close()
}

func consumerFileEvents(ctx context.Context, feeder chan Event) {
	log.Println("Consuming file events")

	for {
		select {
		case event, ok := <-feeder:
			if !ok {
				return
			}

			f := event.Name
			checksum, err := calcChecksum(f)
			if err != nil {
				log.Printf("error calculating checksum of file %s, err: %s", f, err.Error())
				continue
			}

			r := redresserStore[f]
			if r.checksum != checksum {
				log.Println("Correcting file", r.originalPath)
				copy(r.originalPath, r.mountPath)
			} else {
				log.Println("Ignoring event as file is untampered", r.originalPath)
			}

		case <-ctx.Done():
			log.Println("Stopping to consume file events")
			return
		}
	}
}
