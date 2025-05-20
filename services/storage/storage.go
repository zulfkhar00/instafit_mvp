// storage.go
package storage

import "context"

// StorageService defines the methods for interacting with storage providers.
type StorageService interface {
	// UploadBlob uploads binary data and returns a URL or identifier.
	UploadBlob(ctx context.Context, data []byte, filename, contentType string) (string, error)
	// DeleteBlob deletes a blob with a ket from the storage.
	DeleteBlob(ctx context.Context, key string) error
}
