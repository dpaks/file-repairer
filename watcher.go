package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pkg/errors"

	"github.com/fsnotify/fsnotify"
)

func main() {
	rConfig, err := readConfig()
	if err != nil {
		log.Println("error loading file watcher config, err:", err)
		return
	}
	if rConfig.Enabled == false {
		log.Println("File watcher is disabled")
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	// defer cancel()
	feeder := make(chan Event, len(rConfig.WatchList))
	defer close(feeder)

	errChan := make(chan error)
	go consumeFileEvents(ctx, feeder, errChan)
	go filesWatcher(ctx, prepareFileList(rConfig.WatchList), feeder, errChan)

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	select {
	case err = <-errChan:
		log.Println("error starting file watcher, err:", err)
	case <-c:
		log.Println("canceling")
		cancel()
		log.Println("bye")
	}

	time.Sleep(time.Second)
}

func prepareFileList(wl []watchList) []string {
	fl := make([]string, 0)
	for _, wlItem := range wl {
		fl = append(fl, wlItem.MountPath)
	}

	return fl
}

func frameEvent(fileName, op string) Event {
	return Event{
		Name: fileName,
		Op:   op,
	}
}

func fileEventWatcher(ctx context.Context, files []string, feeder chan Event) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return errors.Wrap(err, "failed to start file event watcher")
	}

	log.Println("Starting file watcher")

	go func() {
		defer func() {
			_ = watcher.Close()
			log.Println("Closing file watcher")
		}()

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				if event.Op&(fsnotify.Write|fsnotify.Remove) > 0 {
					log.Printf("fileEventWatcher: modified file '%s' at '%s'\n\n", event.Name, time.Now().String())
					feeder <- frameEvent(event.Name, event.Op.String())
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			case <-ctx.Done():
				return
			}

		}
	}()

	for _, f := range files {
		err = watcher.Add(f)
		if err != nil {
			return errors.Wrap(err, "failed to add file to watcher")
		}
	}

	return nil
}

func filePoller(ctx context.Context, files []string, feeder chan Event) error {
	log.Println("Starting file poller")

	rConfig, err := readConfig()
	if err != nil {
		log.Println("error loading file watcher config, err:", err)
		return err
	}

	log.Println("Polling interval is", rConfig.PollingInterval)
	go func() {
		for _ = range time.NewTicker(rConfig.PollingInterval).C {
			select {
			case <-ctx.Done():
				log.Println("Closing file poller")
				return
			default:
				for _, f := range files {
					checksum, err := calcChecksum(f)
					if err != nil {
						log.Printf("error calculating checksum of file %s, err: %s", f, err.Error())
					}
					r, ok := redresserStore[f]
					if !ok {
						log.Printf("Mount file %s not found in store\n", f)
						continue
					}
					if r.checksum != checksum {
						log.Printf("filePoller: modified file '%s' at '%s'\n\n", f, time.Now().String())
						feeder <- frameEvent(f, "modified")
					}
				}
			}
		}
	}()

	return nil
}

func filesWatcher(ctx context.Context, files []string, feeder chan Event, errChan chan error) {
	if len(files) == 0 {
		log.Println("No files to watch")
		return
	}
	if err := fileEventWatcher(ctx, files, feeder); err != nil {
		errChan <- errors.Wrap(err, "failed to run fileEventWatcher")
	}
	if err := filePoller(ctx, files, feeder); err != nil {
		errChan <- errors.Wrap(err, "failed to run filePoller")
	}

	<-ctx.Done()
}
