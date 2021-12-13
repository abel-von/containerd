package sandbox

import (
	"context"
	"time"

	"github.com/containerd/containerd/mount"
	"github.com/gogo/protobuf/types"
	runtime "github.com/opencontainers/runtime-spec/specs-go"
)

type Sandbox struct {
	ID          string
	Spec        *runtime.Spec
	Containers  []Container
	TaskAddress string
	Labels      map[string]string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Extensions  map[string]types.Any
}

// Status is a structure describing current sandbox instance status
type Status struct {
	// ID is a sandbox identifier
	ID string
	// PID is a process ID of the sandbox process (if any)
	PID uint32
	// State is current sandbox state
	State State
	// Version represents sandbox version
	Version string
	// Extra is additional information that might be included by sandbox implementations
	Extra map[string]types.Any
}

// State is current state of a sandbox (reported by `Status` call)
// TODO the state should be a enum in sandbox.proto,
// otherwise sandbox plugins don't know how to set the state
type State string

const (
	StateStarting State = "starting"
	StateReady    State = "ready"
	StateStopping State = "stopping"
	StateStopped  State = "stopped"
	StatePaused   State = "paused"
)

type Container struct {
	ID         string
	Spec       *types.Any
	Rootfs     []mount.Mount
	Io         *IO
	Processes  []Process
	Bundle     string
	Labels     map[string]string
	Extensions map[string]types.Any
}

type Process struct {
	Id         string
	Io         *IO
	Process    *types.Any
	Extensions map[string]types.Any
}

type IO struct {
	Stdin    string
	Stdout   string
	Stderr   string
	Terminal bool
}

// Sandboxer defines the methods required to implement a sandbox sandboxer for
// creating, updating and deleting sandboxes.
type Sandboxer interface {
	Create(ctx context.Context, sb *Sandbox) (*Sandbox, error)
	Update(ctx context.Context, sb *Sandbox, fieldpaths ...string) (*Sandbox, error)
	AppendContainer(ctx context.Context, sandboxId string, container *Container) (*Container, error)
	UpdateContainer(ctx context.Context, sandboxId string, container *Container) (*Container, error)
	RemoveContainer(ctx context.Context, sandboxId string, id string) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, fields ...string) ([]Sandbox, error)
	Get(ctx context.Context, id string) (*Sandbox, error)
	Status(ctx context.Context, id string) (Status, error)
}

// Controller is the interface to be implemented by sandbox plugins to manage sandbox instances.
type Controller interface {
	// Start creates and runs a new sandbox instance.
	// Clients may configure sandbox environment via runtime spec, labels, and extensions.
	Start(ctx context.Context, sandbox *Sandbox) (*Sandbox, error)
	// Shutdown shuts down a sandbox instance identified by id.
	Shutdown(ctx context.Context, id string) error
	// Pause pauses a running sandbox, all running containers in the sandbox will be paused
	Pause(ctx context.Context, id string) error
	// Resume resumes a paused sandbox
	Resume(ctx context.Context, id string) error
	// Update updates a sandbox metadata, such as spec, labels, and extensions.
	Update(ctx context.Context, sandboxId string, sandbox *Sandbox) (*Sandbox, error)
	// AppendContainer append the container resources to the sandbox
	AppendContainer(ctx context.Context, sandboxId string, container *Container) (*Container, error)
	// UpdateContainer update the container resources in the sandbox, like updating cpu/mem resources,
	// or add/remove resources(especially io pipes) of an exec process
	UpdateContainer(ctx context.Context, sandboxId string, container *Container) (*Container, error)
	// RemoveContainer remove the container resources in the sandbox
	RemoveContainer(ctx context.Context, sandboxId string, id string) error
	// Status returns a status of a sandbox identified by id.
	Status(ctx context.Context, id string) (Status, error)
	// Ping pings the running sandbox, return error if ping failed.
	Ping(ctx context.Context, id string) error
}
