package huelio

import (
	"encoding/json"
	"log"
	"os"
	"sync"

	"github.com/amimof/huego"
	"github.com/pkg/errors"
)

// Store reads and writes credentials
type Store interface {
	// Read reads credentials from this store.
	// When no credentials exist should return nil, nil.
	Read() (*Credentials, error)

	// Write writes credentials to this store.
	// Write(nil) deletes any stored credentials.
	// When the credentials store is read-only, should return ErrStoreReadOnly
	Write(credentials *Credentials) error
}

var ErrStoreReadOnly = errors.New("Store: store is readonly")

// Credentials represents credentials to a hue bridge.
type Credentials struct {
	Hostname string `json:"hostname"`
	Username string `json:"username"`
}

type JSONFileStore string

func (f JSONFileStore) Read() (*Credentials, error) {
	h, err := os.Open(string(f))
	if err != nil {
		if os.IsNotExist(err) { // nothing stored
			return nil, nil
		}
		return nil, err
	}
	defer h.Close()

	decoder := json.NewDecoder(h)

	credentials := &Credentials{}
	err = decoder.Decode(credentials)
	return credentials, err
}

func (f JSONFileStore) Write(credentials *Credentials) error {
	// delete the credentials (if any)
	if credentials == nil {
		err := os.Remove(string(f))
		if err != nil && os.IsNotExist(err) {
			return nil
		}
		return err
	}

	// create the file (if it doesn't exist)
	h, err := os.OpenFile(string(f), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer h.Close()

	// and encode!
	return json.NewEncoder(h).Encode(credentials)
}

type InMemoryStore struct {
	Readonly    bool
	credentials *Credentials
}

func (store *InMemoryStore) Read() (*Credentials, error) {
	return store.credentials, nil
}

func (store *InMemoryStore) Write(credentials *Credentials) error {
	if store.Readonly {
		return ErrStoreReadOnly
	}
	store.credentials = credentials
	return nil
}

// Finder finds new credentials
type Finder interface {
	Find() (*Credentials, error)
}

type PartialFinder struct {
	Logger *log.Logger

	NewName string

	Hostname string
	Username string
}

func (pf PartialFinder) Find() (*Credentials, error) {
	if pf.Hostname != "" && pf.Username != "" {
		if pf.Logger != nil {
			pf.Logger.Println("PartialFinder: Returning fixed credentials")
		}

		return &Credentials{
			Hostname: pf.Hostname,
			Username: pf.Username,
		}, nil
	}

	var bridge *huego.Bridge
	if pf.Hostname == "" {
		if pf.Logger != nil {
			pf.Logger.Printf("PartialFinder: Discovering bridge")
		}

		var err error
		bridge, err = huego.Discover()
		if err != nil {
			return nil, errors.Wrap(err, "Unable to connect to bridge")
		}

		if pf.Logger != nil {
			pf.Logger.Printf("PartialFinder: Found bridge at %s", bridge.Host)
		}
	} else {
		bridge = huego.New(pf.Hostname, "")

		if pf.Logger != nil {
			pf.Logger.Printf("PartialFinder: Connecting to existing bridge at %s", bridge.Host)
		}
	}

	if pf.Logger != nil {
		pf.Logger.Printf("PartialFinder: Creating new user %s", pf.NewName)
	}

	user, err := bridge.CreateUser(pf.NewName)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create new user")
	}

	if pf.Logger != nil {
		pf.Logger.Printf("PartialFinder: Received new user %s", user)
	}

	return &Credentials{
		Hostname: bridge.Host,
		Username: user,
	}, nil
}

// StoreManager manages a store
type StoreManager struct {
	l sync.Mutex

	Finder Finder
	Store  Store
}

// Connect connects to a hue bridge
func (sm *StoreManager) Connect() (*huego.Bridge, error) {
	sm.l.Lock()
	defer sm.l.Unlock()

	// if we have credentials in the store, return them!
	creds, err := sm.Store.Read()
	if err != nil {
		return nil, errors.Wrap(err, "unable to read stored credentials")
	}
	if creds != nil {
		return NewBridge(creds)
	}

	// create new credentials
	credentials, err := sm.Finder.Find()
	if err != nil {
		return nil, errors.Wrap(err, "unable to generate new credentials")
	}

	// make a bridge
	bridge, err := NewBridge(credentials)
	if err != nil {
		return nil, errors.Wrap(err, "bridge connection failed")
	}

	// write the credentials to the store, unless it is readonly
	if err := sm.Store.Write(credentials); err != nil && err != ErrStoreReadOnly {
		return nil, errors.Wrap(err, "Unable to store credentials")
	}

	// and return!
	return bridge, nil
}

// NewBridge creates a new bridge based on credentials
func NewBridge(credentials *Credentials) (*huego.Bridge, error) {
	bridge := huego.New(credentials.Hostname, credentials.Username)
	_, err := bridge.GetCapabilities()
	if err != nil {
		return nil, err
	}
	return bridge, nil
}
