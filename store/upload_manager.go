package store

import (
	"sync"

	desc "github.com/iterum-provenance/iterum-go/descriptors"
	"github.com/iterum-provenance/iterum-go/minio"
	"github.com/iterum-provenance/iterum-go/transmit"
	"github.com/iterum-provenance/sidecar/env/config"
	"github.com/iterum-provenance/sidecar/garbage"

	"github.com/prometheus/common/log"
)

// UploadManager is the structure that consumes LocalFragmentDesc structures and uploads them to minio
type UploadManager struct {
	ToUpload       chan transmit.Serializable // desc.LocalFragmentDesc
	Completed      chan transmit.Serializable // desc.RemoteFragmentDesc
	Minio          minio.Config
	sidecarConfig  *config.Config
	fragments      int
	strictOrdering bool
	fragCollector  garbage.FragmentCollector
}

// NewUploadManager creates a new upload manager and initiates a client of the Minio service
func NewUploadManager(toUpload, uploaded chan transmit.Serializable,
	sidecarConfig *config.Config, collector garbage.FragmentCollector) UploadManager {

	minio := minio.NewMinioConfigFromEnv() // defaults to an upload setup
	if err := minio.Connect(); err != nil {
		log.Fatal(err)
	}
	return UploadManager{
		toUpload,
		uploaded,
		minio,
		sidecarConfig,
		0,
		false,
		collector,
	}
}

// StartBlocking enters an endless loop consuming LocalFragmentDesc and uploading the associated data
func (um UploadManager) StartBlocking() {
	var wg sync.WaitGroup
	for {
		msg, ok := <-um.ToUpload
		if !ok {
			break
		}
		uloader := NewUploader(*msg.(*desc.LocalFragmentDesc), um.Minio, um.Completed, um.sidecarConfig, um.fragCollector)

		if um.strictOrdering {
			uloader.StartBlocking()
		} else {
			uloader.Start(&wg)
		}

		um.fragments++
	}
	wg.Wait()
	um.Stop()
}

// Start asychronously calls StartBlocking via a Goroutine
func (um UploadManager) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		um.StartBlocking()
	}()
}

// Stop finishes up and notifies the user of its progress
func (um UploadManager) Stop() {
	log.Infof("UploadManager finishing up, (tried to) upload(ed) %v fragments", um.fragments)
	close(um.Completed)
	close(um.fragCollector.Track)
	close(um.fragCollector.Collect)
}
