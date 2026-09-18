package hub

import "errors"

var (
	// ErrHubNotFound is returned when a hub does not exist in the registry.
	ErrHubNotFound = errors.New("hub not found")

	// ErrHubOffline is returned when trying to communicate with an offline hub.
	ErrHubOffline = errors.New("hub is offline")

	// ErrDuplicateHub is returned when attempting to register a hub
	// with an ID that already exists.
	ErrDuplicateHub = errors.New("hub already registered")

	// ErrInvalidPublicKey is returned when the hub's public key is malformed.
	ErrInvalidPublicKey = errors.New("invalid hub public key")
)
