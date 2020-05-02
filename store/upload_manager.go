package store

import (
	"sync"

	desc "github.com/iterum-provenance/iterum-go/descriptors"
	"github.com/iterum-provenance/iterum-go/minio"
	"github.com/prometheus/common/log"

	"github.com/iterum-provenance/iterum-go/transmit"
)

// UploadManager is the structure that consumes LocalFragmentDesc structures and uploads them to minio
type UploadManager struct {
	ToUpload  chan transmit.Serializable // desc.LocalFragmentDesc
	Completed chan transmit.Serializable // desc.RemoteFragmentDesc
	Minio     minio.Config
	fragments int
}

// NewUploadManager creates a new upload manager and initiates a client of the Minio service
func NewUploadManager(minio minio.Config, toUpload, completed chan transmit.Serializable) UploadManager {

	return UploadManager{toUpload, completed, minio, 0}
}

// StartBlocking enters an endless loop consuming LocalFragmentDesc and uploading the associated data
func (um UploadManager) StartBlocking() {
	var wg sync.WaitGroup
	for {
		msg, ok := <-um.ToUpload
		if !ok {
			break
		}

		uloader := NewUploader(*msg.(*desc.LocalFragmentDesc), um.Minio, um.Completed)
		uloader.Start(&wg)
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
}
