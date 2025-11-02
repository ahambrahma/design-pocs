package main

import (
	"errors"
	"sync/atomic"
)

type BackendServerProperties struct {
	id                     string
	baseUrl                string
	apiPort                uint
	healthCheckPort        uint
	activeConnectionsCount atomic.Int32
	weight                 uint
}

func NewBackendServer(id string, url string, apiPort uint, healthCheckPort uint) *BackendServerProperties {
	return &BackendServerProperties{
		id:                     id,
		baseUrl:                url,
		apiPort:                apiPort,
		healthCheckPort:        healthCheckPort,
		activeConnectionsCount: atomic.Int32{},
	}
}

func (b *BackendServerProperties) GetID() string {
	return b.id
}

func (b *BackendServerProperties) GetUrl() string {
	return b.baseUrl
}

func (b *BackendServerProperties) GetAPIPort() uint {
	return b.apiPort
}

func (b *BackendServerProperties) SetWeight(weight uint) error {
	if weight > 100 {
		return errors.New("Weight cannot be greater than 100")
	}

	b.weight = weight
	return nil
}

func (b *BackendServerProperties) IncrementConnections() {
	b.activeConnectionsCount.Add(1)
}

func (b *BackendServerProperties) DecrementConnections() {
	b.activeConnectionsCount.Add(-1)
}

func (b *BackendServerProperties) GetActiveConnections() int32 {
	return b.activeConnectionsCount.Load()
}
