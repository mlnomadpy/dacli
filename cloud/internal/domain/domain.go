// Package domain contains stable concepts shared by the API and worker.
package domain

const (
	ContractVersion = 1
	ServiceName     = "dacli-control-plane"
)

// Component identifies a process without implying an unimplemented feature.
type Component string

const (
	ComponentAPI    Component = "api"
	ComponentWorker Component = "worker"
)
