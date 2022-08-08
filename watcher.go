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
	rConfig := readConfig()
	if rConfig.Enabled == false {
		log.Println("File watcher is disabled")
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	// defer cancel()
	feeder := make(chan Event, len(rConfig.WatchList))
	defer close(feeder)

	go consumeFileEvents(ctx, feeder)
	go filesWatcher(ctx, prepareFileList(rConfig.WatchList), feeder)

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	log.Println("canceling")
	cancel()
	log.Println("bye")
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

	defer func() {
		_ = watcher.Close()
		log.Println("Closing file watcher")
	}()

	log.Println("Starting file watcher")

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				if event.Op&(fsnotify.Write|fsnotify.Remove) > 0 {
					log.Printf("modified file '%s' at '%s'\n\n", event.Name, time.Now().String())
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

	<-ctx.Done()

	return nil
}

func filePoller(ctx context.Context, files []string, feeder chan Event) error {
	return nil
}

func filesWatcher(ctx context.Context, files []string, feeder chan Event) {
	if len(files) == 0 {
		log.Println("No files to watch")
		return
	}
	if err := fileEventWatcher(ctx, files, feeder); err != nil {
		panic(err)
	}
	if err := filePoller(ctx, files, feeder); err != nil {
		panic(err)
	}

}
