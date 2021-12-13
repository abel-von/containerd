/*
   Copyright The containerd Authors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package sandbox

import (
	api "github.com/containerd/containerd/api/services/sandbox/v1"
	apiTypes "github.com/containerd/containerd/api/types"
	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/mount"
	"github.com/containerd/typeurl"
	"github.com/gogo/protobuf/types"
	runtime "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/pkg/errors"
)

func StatusFromProto(proto *api.Status) Status {
	return Status{
		ID:      proto.ID,
		PID:     proto.Pid,
		State:   State(proto.State),
		Version: proto.Version,
		Extra:   proto.Extra,
	}
}

func StatusToProto(status Status) *api.Status {
	return &api.Status{
		ID:      status.ID,
		Pid:     status.PID,
		State:   string(status.State),
		Version: status.Version,
		Extra:   status.Extra,
	}
}

func ContainerFromProto(proto api.Container) (Container, error) {
	var processes []Process
	for _, proc := range proto.Processes {
		processes = append(processes, Process{
			Id:         proc.ID,
			Io:         IOFromProto(proc.Io),
			Process:    proc.Process,
			Extensions: proc.Extra,
		})
	}
	return Container{
		ID:         proto.ID,
		Spec:       proto.Spec,
		Io:         IOFromProto(proto.Io),
		Rootfs:     mountsFromProto(proto.Rootfs),
		Bundle:     proto.Bundle,
		Processes:  processes,
		Labels:     proto.Labels,
		Extensions: proto.Extensions,
	}, nil
}

func ContainerToProto(container Container) (api.Container, error) {
	var processes []api.Process
	for _, proc := range container.Processes {
		processes = append(processes, api.Process{
			ID:      proc.Id,
			Io:      IOToProto(proc.Io),
			Process: proc.Process,
			Extra:   proc.Extensions,
		})
	}
	return api.Container{
		ID:         container.ID,
		Spec:       container.Spec,
		Io:         IOToProto(container.Io),
		Rootfs:     mountsToProto(container.Rootfs),
		Bundle:     container.Bundle,
		Processes:  processes,
		Labels:     container.Labels,
		Extensions: container.Extensions,
	}, nil
}

func mountsFromProto(proto []apiTypes.Mount) []mount.Mount {
	var mounts []mount.Mount

	for _, m := range proto {
		mount := mount.Mount{
			Type:    m.Type,
			Source:  m.Source,
			Options: m.Options,
		}
		mounts = append(mounts, mount)
	}
	return mounts
}

func mountsToProto(mounts []mount.Mount) []apiTypes.Mount {
	var protos []apiTypes.Mount

	for _, m := range mounts {
		mount := apiTypes.Mount{
			Type:    m.Type,
			Source:  m.Source,
			Options: m.Options,
		}
		protos = append(protos, mount)
	}
	return protos
}

func IOFromProto(proto *api.IO) *IO {
	if proto == nil {
		return nil
	}
	return &IO{
		Stdin:    proto.Stdin,
		Stdout:   proto.Stdout,
		Stderr:   proto.Stderr,
		Terminal: proto.Terminal,
	}
}

func IOToProto(io *IO) *api.IO {
	if io == nil {
		return nil
	}
	return &api.IO{
		Stdin:    io.Stdin,
		Stdout:   io.Stdout,
		Stderr:   io.Stderr,
		Terminal: io.Terminal,
	}
}

func SandboxFromProto(proto *api.Sandbox) (*Sandbox, error) {
	if proto == nil {
		return nil, nil
	}
	spec, err := anyToSpec(proto.Spec)
	if err != nil {
		return nil, err
	}
	containers, err := containersFromProto(proto.Containers)
	if err != nil {
		return nil, err
	}
	return &Sandbox{
		ID:          proto.ID,
		Spec:        spec,
		Containers:  containers,
		TaskAddress: proto.TaskAddress,
		Labels:      proto.Labels,
		CreatedAt:   proto.CreatedAt,
		UpdatedAt:   proto.UpdatedAt,
		Extensions:  proto.Extensions,
	}, nil
}

func SandboxToProto(sandbox *Sandbox) (*api.Sandbox, error) {
	if sandbox == nil {
		return nil, nil
	}
	spec, err := toAny(sandbox.Spec)
	if err != nil {
		return nil, err
	}
	containers, err := containersToProto(sandbox.Containers)
	if err != nil {
		return nil, err
	}
	return &api.Sandbox{
		ID:          sandbox.ID,
		Spec:        spec,
		Containers:  containers,
		TaskAddress: sandbox.TaskAddress,
		Labels:      sandbox.Labels,
		CreatedAt:   sandbox.CreatedAt,
		UpdatedAt:   sandbox.UpdatedAt,
		Extensions:  sandbox.Extensions,
	}, nil
}

func containersFromProto(proto []api.Container) ([]Container, error) {
	var containers []Container
	for _, c := range proto {
		container, err := ContainerFromProto(c)
		if err != nil {
			return containers, err
		}
		containers = append(containers, container)
	}
	return containers, nil
}

func containersToProto(proto []Container) ([]api.Container, error) {
	var containers []api.Container
	for _, c := range proto {
		any, err := ContainerToProto(c)
		if err != nil {
			return containers, err
		}
		containers = append(containers, any)
	}
	return containers, nil
}

func toAny(v interface{}) (*types.Any, error) {
	any, err := typeurl.MarshalAny(v)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal struct to any")
	}

	return any, nil
}

func fromAny(any *types.Any) (interface{}, error) {
	if any == nil {
		return nil, errors.Wrap(errdefs.ErrInvalidArgument, "any can't be empty")
	}

	raw, err := typeurl.UnmarshalAny(any)
	if err != nil {
		return nil, errors.Wrapf(errdefs.ErrInvalidArgument, "failed to unmarshal any: %v", err)
	}

	return raw, nil
}

func anyToSpec(any *types.Any) (*runtime.Spec, error) {
	raw, err := fromAny(any)
	if err != nil {
		return nil, err
	}

	spec, ok := raw.(*runtime.Spec)
	if !ok {
		return nil, errors.Wrap(errdefs.ErrInvalidArgument, "unexpected sandbox spec structure")
	}

	return spec, nil
}
