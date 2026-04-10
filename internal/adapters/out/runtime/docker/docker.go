package docker

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/kleffio/kleff-daemon/internal/application/ports"
	"github.com/kleffio/kleff-daemon/pkg/labels"
)

const labelPrefix = "kleff."

// Adapter is a Docker RuntimeAdapter.
// All three strategies (agones, statefulset, deployment) map to the same
// Docker container lifecycle — the strategy hint is ignored here.
type Adapter struct {
	client *client.Client
	nodeID string
}

func New(nodeID string) (*Adapter, error) {
	c, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}
	return &Adapter{client: c, nodeID: nodeID}, nil
}

// Deploy pulls the image and starts a new container.
func (a *Adapter) Deploy(ctx context.Context, spec ports.WorkloadSpec) (*ports.RunningServer, error) {
	// Pull image.
	rc, err := a.client.ImagePull(ctx, spec.Image, image.PullOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to pull image %s: %w", spec.Image, err)
	}
	rc.Close()

	containerID, err := a.createContainer(ctx, spec)
	if err != nil {
		return nil, err
	}

	if err := a.client.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	return &ports.RunningServer{
		Labels: labels.WorkloadLabels{
			OwnerID: spec.OwnerID, ServerID: spec.ServerID,
			BlueprintID: spec.BlueprintID, NodeID: a.nodeID,
		},
		RuntimeRef: containerID,
		State:      "Running",
	}, nil
}

// Start restarts a stopped container. If it no longer exists, re-creates it.
func (a *Adapter) Start(ctx context.Context, spec ports.WorkloadSpec) (*ports.RunningServer, error) {
	containerID, err := a.findContainer(ctx, spec.ServerID)
	if err != nil {
		// Container gone — re-create it.
		return a.Deploy(ctx, spec)
	}

	if err := a.client.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	return &ports.RunningServer{RuntimeRef: containerID, State: "Running"}, nil
}

// Stop stops the container without removing it.
func (a *Adapter) Stop(ctx context.Context, workloadID string) error {
	containerID, err := a.findContainer(ctx, workloadID)
	if err != nil {
		return err
	}
	timeout := 10
	if err := a.client.ContainerStop(ctx, containerID, container.StopOptions{Timeout: &timeout}); err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}
	return nil
}

// Remove stops and removes the container.
func (a *Adapter) Remove(ctx context.Context, workloadID string) error {
	containerID, err := a.findContainer(ctx, workloadID)
	if err != nil {
		return err
	}
	timeout := 10
	_ = a.client.ContainerStop(ctx, containerID, container.StopOptions{Timeout: &timeout})
	if err := a.client.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: true}); err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}
	return nil
}

// Status returns the current state of the container.
func (a *Adapter) Status(ctx context.Context, workloadID string) (*ports.WorkloadHealth, error) {
	containerID, err := a.findContainer(ctx, workloadID)
	if err != nil {
		return nil, err
	}
	info, err := a.client.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}
	state := strings.ToLower(info.State.Status)
	return &ports.WorkloadHealth{WorkloadID: workloadID, State: state}, nil
}

// Endpoint returns the first exposed host port.
func (a *Adapter) Endpoint(ctx context.Context, workloadID string) (string, error) {
	containerID, err := a.findContainer(ctx, workloadID)
	if err != nil {
		return "", err
	}
	info, err := a.client.ContainerInspect(ctx, containerID)
	if err != nil {
		return "", fmt.Errorf("failed to inspect container: %w", err)
	}
	for _, bindings := range info.NetworkSettings.Ports {
		if len(bindings) > 0 {
			return fmt.Sprintf("127.0.0.1:%s", bindings[0].HostPort), nil
		}
	}
	return "", fmt.Errorf("no exposed ports found for workload %s", workloadID)
}

// Logs streams the container's stdout/stderr.
func (a *Adapter) Logs(ctx context.Context, workloadID string, follow bool) (io.ReadCloser, error) {
	containerID, err := a.findContainer(ctx, workloadID)
	if err != nil {
		return nil, err
	}
	rc, err := a.client.ContainerLogs(ctx, containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     follow,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get logs: %w", err)
	}
	return rc, nil
}

// --- Helpers ---

func (a *Adapter) createContainer(ctx context.Context, spec ports.WorkloadSpec) (string, error) {
	env := make([]string, 0, len(spec.EnvOverrides))
	for k, v := range spec.EnvOverrides {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	exposedPorts := nat.PortSet{}
	portBindings := nat.PortMap{}
	for _, p := range spec.PortRequirements {
		proto := strings.ToLower(p.Protocol)
		if proto == "" {
			proto = "tcp"
		}
		natPort := nat.Port(fmt.Sprintf("%d/%s", p.TargetPort, proto))
		exposedPorts[natPort] = struct{}{}
		portBindings[natPort] = []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: "0"}} // 0 = random host port
	}

	containerLabels := map[string]string{
		labelPrefix + "server_id":    spec.ServerID,
		labelPrefix + "owner_id":     spec.OwnerID,
		labelPrefix + "blueprint_id": spec.BlueprintID,
		labelPrefix + "node_id":      a.nodeID,
	}

	resp, err := a.client.ContainerCreate(ctx,
		&container.Config{
			Image:        spec.Image,
			Env:          env,
			ExposedPorts: exposedPorts,
			Labels:       containerLabels,
		},
		&container.HostConfig{
			PortBindings: portBindings,
		},
		&network.NetworkingConfig{},
		nil,
		spec.ServerID,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}
	return resp.ID, nil
}

// findContainer looks up a container by the kleff server_id label.
func (a *Adapter) findContainer(ctx context.Context, serverID string) (string, error) {
	containers, err := a.client.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: filters.NewArgs(filters.Arg("label", labelPrefix+"server_id="+serverID)),
	})
	if err != nil {
		return "", fmt.Errorf("failed to list containers: %w", err)
	}
	if len(containers) == 0 {
		return "", fmt.Errorf("container not found for server %s", serverID)
	}
	return containers[0].ID, nil
}
