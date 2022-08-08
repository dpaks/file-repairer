package main

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

// var config *viper.Viper

var configYaml = []byte(`
file-watcher:
  enabled: true
  watchlist:
    - mountpath: mount-test.txt
      originalpath: original-test.txt
`)

type watchList struct {
	MountPath    string
	OriginalPath string
}

type redresserConfig struct {
	Enabled   bool
	WatchList []watchList `json:"watchlist"`
}

func readConfig() *redresserConfig {
	config := viper.New()
	config.SetConfigType("yaml")
	err := config.ReadConfig(bytes.NewBuffer(configYaml))
	if err != nil {
		panic(err)
	}

	config = config.Sub("file-watcher")
	rConfig := new(redresserConfig)
	err = config.Unmarshal(rConfig)
	if err != nil {
		panic(err)
	}

	return rConfig
}

var redresserStore map[string]redresser

type redresser struct {
	mountPath    string
	originalPath string
	checksum     string
}

func init() {
	log.Println("gobbler.init()")
	rConfig := readConfig()

	redresserStore = make(map[string]redresser)
	for _, watchListItem := range rConfig.WatchList {
		mountPath, originalPath := watchListItem.MountPath, watchListItem.OriginalPath
		r := redresser{
			mountPath:    mountPath,
			originalPath: originalPath,
			checksum:     calcChecksumPanic(originalPath),
		}
		redresserStore[mountPath] = r
		log.Printf("Calculated checksum for %s as %s\n", r.originalPath, r.checksum)
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

func consumeFileEvents(ctx context.Context, feeder chan Event) {
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
				log.Println("Correcting file", r.mountPath)
				copy(r.originalPath, r.mountPath)
			} else {
				log.Println("Ignoring event as file is untampered", r.mountPath)
			}

		case <-ctx.Done():
			log.Println("Stopping to consume file events")
			return
		}
	}
}
