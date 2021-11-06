package creds

import (
	"encoding/json"
	"os"
)

// Store reads and writes credentials
type Store interface {
	// Read reads credentials from this store.
	// When no credentials exist, returns nil.
	Read() (*Credentials, error)

	// Write writes credentials to this store.
	// Write(nil) deletes any stored credentials
	Write(credentials *Credentials) error
}

// JSONFileStore stores credentials in the provided JSON file on disk.
// Implements Store.
type JSONFileStore string

// Read reads credentials from the provided filename on disk.
//
// When the file does not exist, the store is considered empty.
// When the file cannot be considered, or contains data other than credentials, this is considered an error.
func (f JSONFileStore) Read() (*Credentials, error) {
	h, err := os.Open(string(f))
	if err != nil {
		// file does not exist, meaning the store it empty
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer h.Close()

	// read json from the file
	credentials := &Credentials{}
	err = json.NewDecoder(h).Decode(credentials)
	return credentials, err
}

// Write writes credentials to the provided filename on disk.
//
// When creating a new file, uses chmod 0600 to prevent other users from accessing the file.
// When the credentials being written are empty, deletes the file.
func (f JSONFileStore) Write(credentials *Credentials) error {
	// delete the credentials
	if credentials == nil {
		err := os.Remove(string(f))
		if err != nil && os.IsNotExist(err) {
			return nil
		}
		return err
	}

	// create a new file
	h, err := os.OpenFile(string(f), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer h.Close()

	// encode the credentials!
	return json.NewEncoder(h).Encode(credentials)
}

// InMemoryStore stores credentials in-memory.
// It implements Store.
type InMemoryStore struct {
	credentials *Credentials
}

// Read reads credentials from memory
func (store *InMemoryStore) Read() (*Credentials, error) {
	return store.credentials, nil
}

// Write writes credentials to memory
func (store *InMemoryStore) Write(credentials *Credentials) error {
	store.credentials = credentials
	return nil
}
