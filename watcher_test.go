package main

import (
	"context"
	"reflect"
	"testing"
	"time"
)

func Test_prepareFileList(t *testing.T) {
	wl := []watchList{
		{
			MountPath:    "mount file1",
			OriginalPath: "original file1",
		},
		{
			MountPath:    "mount file2",
			OriginalPath: "original file2",
		},
	}
	want := []string{"mount file1", "mount file2"}
	if got := prepareFileList(wl); !reflect.DeepEqual(got, want) {
		t.Errorf("prepareFileList() = %v, want %v", got, want)
	}
}

func Test_filePoller(t *testing.T) {
	type args struct {
		ctx    context.Context
		files  []string
		feeder chan Event
	}
	tests := []struct {
		name    string
		args    args
		config  []byte
		wantErr bool
	}{
		{
			name: "sanity",
			args: args{
				ctx:    context.Background(),
				files:  []string{"testfile"},
				feeder: make(chan Event, 1),
			},
			wantErr: false,
			config: []byte(`
file_watcher:
  enabled: true
  polling_interval: 1s
  watchlist:
    - mountpath: mount-test.txt
      originalpath: original-test.txt
`),
		},
		{
			name: "process checksum",
			args: args{
				ctx:    context.Background(),
				files:  []string{"testfile"},
				feeder: make(chan Event, 1),
			},
			wantErr: false,
			config: []byte(`
file_watcher:
  enabled: true
  polling_interval: 1h
  watchlist:
    - mountpath: mount-test.txt
      originalpath: original-test.txt
`),
		},
	}
	redresserStore = map[string]redresser{
		"testfile": {
			mountPath:    "mount",
			originalPath: "original",
			checksum:     "xyz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configYaml = tt.config
			if err := filePoller(tt.args.ctx, tt.args.files, tt.args.feeder); (err != nil) != tt.wantErr {
				t.Errorf("filePoller() error = %v, wantErr %v", err, tt.wantErr)
			}
			time.Sleep(time.Second)
		})
	}
}
