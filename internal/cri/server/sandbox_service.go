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

package server

import (
	"context"
	"fmt"
	"time"

	"github.com/containerd/platforms"

	"github.com/containerd/containerd/v2/client"
	"github.com/containerd/containerd/v2/core/sandbox"
	criconfig "github.com/containerd/containerd/v2/internal/cri/config"
)

type criSandboxService struct {
	sandboxers map[string]sandbox.Controller
	config     *criconfig.Config
}

func newCriSandboxService(config *criconfig.Config, sandboxers map[string]sandbox.Controller) *criSandboxService {
	return &criSandboxService{
		sandboxers: sandboxers,
		config:     config,
	}
}

func (c *criSandboxService) SandboxController(sandboxer string) (sandbox.Controller, error) {
	sbController, ok := c.sandboxers[sandboxer]
	if !ok {
		return nil, fmt.Errorf("failed to get sandbox controller by %s", sandboxer)
	}
	return sbController, nil
}

func (c *criSandboxService) CreateSandbox(ctx context.Context, info sandbox.Sandbox, opts ...sandbox.CreateOpt) error {
	sbController, ok := c.sandboxers[info.Sandboxer]
	if !ok {
		return fmt.Errorf("failed to get sandbox controller by %s", info.Sandboxer)
	}
	return sbController.Create(ctx, info, opts...)
}

func (c *criSandboxService) StartSandbox(ctx context.Context, sandboxer string, sandboxID string) (sandbox.ControllerInstance, error) {
	sbController, ok := c.sandboxers[sandboxer]
	if !ok {
		return sandbox.ControllerInstance{}, fmt.Errorf("failed to get sandbox controller by %s", sandboxer)
	}
	return sbController.Start(ctx, sandboxID)
}

func (c *criSandboxService) WaitSandbox(ctx context.Context, sandboxer string, sandboxID string) (<-chan client.ExitStatus, error) {
	sbController, ok := c.sandboxers[sandboxer]
	if !ok {
		return nil, fmt.Errorf("failed to get sandbox controller by %s", sandboxer)
	}

	ch := make(chan client.ExitStatus, 1)
	go func() {
		defer close(ch)

		exitStatus, err := sbController.Wait(ctx, sandboxID)
		if err != nil {
			ch <- *client.NewExitStatus(client.UnknownExitStatus, time.Time{}, err)
			return
		}

		ch <- *client.NewExitStatus(exitStatus.ExitStatus, exitStatus.ExitedAt, nil)
	}()

	return ch, nil
}

func (c *criSandboxService) SandboxStatus(ctx context.Context, sandboxer string, sandboxID string, verbose bool) (sandbox.ControllerStatus, error) {
	sbController, ok := c.sandboxers[sandboxer]
	if !ok {
		return sandbox.ControllerStatus{}, fmt.Errorf("failed to get sandbox controller by %s", sandboxer)
	}
	return sbController.Status(ctx, sandboxID, verbose)
}

func (c *criSandboxService) SandboxPlatform(ctx context.Context, sandboxer string, sandboxID string) (platforms.Platform, error) {
	sbController, ok := c.sandboxers[sandboxer]
	if !ok {
		return platforms.Platform{}, fmt.Errorf("failed to get sandbox controller by %s", sandboxer)
	}
	return sbController.Platform(ctx, sandboxID)
}

func (c *criSandboxService) ShutdownSandbox(ctx context.Context, sandboxer string, sandboxID string) error {
	sbController, ok := c.sandboxers[sandboxer]
	if !ok {
		return fmt.Errorf("failed to get sandbox controller by %s", sandboxer)
	}
	return sbController.Shutdown(ctx, sandboxID)
}

func (c *criSandboxService) StopSandbox(ctx context.Context, sandboxer, sandboxID string, opts ...sandbox.StopOpt) error {
	sbController, ok := c.sandboxers[sandboxer]
	if !ok {
		return fmt.Errorf("failed to get sandbox controller by %s", sandboxer)
	}
	return sbController.Stop(ctx, sandboxID, opts...)
}
