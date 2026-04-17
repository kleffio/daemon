package docker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	dnet "github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/kleffio/kleff-daemon/internal/application/ports"
	"github.com/kleffio/kleff-daemon/pkg/labels"
)

// Adapter is a Docker RuntimeAdapter.
// All three strategies (agones, statefulset, deployment) map to the same
// Docker container lifecycle — the strategy hint is ignored here.
type Adapter struct {
	client *client.Client
	nodeID string
}

var errContainerNotFound = errors.New("container not found")

func New(nodeID string) (*Adapter, error) {
	c, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}
	return &Adapter{client: c, nodeID: nodeID}, nil
}

// Ping checks if the Docker daemon is reachable.
func (a *Adapter) Ping(ctx context.Context) error {
	_, err := a.client.Ping(ctx)
	if err != nil {
		return fmt.Errorf("docker daemon unreachable: %w", err)
	}
	return nil
}

// EnsureProjectScope creates the per-project bridge network if it does not
// already exist. The network is the isolation boundary between projects — all
// containers in a project attach to it, and nothing else can reach them.
func (a *Adapter) EnsureProjectScope(ctx context.Context, projectID, projectSlug string) (*ports.ProjectScope, error) {
	if projectID == "" {
		return nil, fmt.Errorf("project id is required")
	}
	name := projectNetworkName(projectID)

	existing, err := a.client.NetworkList(ctx, dnet.ListOptions{
		Filters: filters.NewArgs(filters.Arg("label", labels.ProjectID+"="+projectID)),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list networks: %w", err)
	}
	if len(existing) > 0 {
		return &ports.ProjectScope{
			ProjectID:   projectID,
			ProjectSlug: projectSlug,
			NetworkName: existing[0].Name,
		}, nil
	}

	netLabels := map[string]string{
		labels.ManagedBy: labels.ManagedByValue,
		labels.ProjectID: projectID,
		labels.NodeID:    a.nodeID,
	}
	if projectSlug != "" {
		netLabels[labels.ProjectSlug] = projectSlug
	}
	if _, err := a.client.NetworkCreate(ctx, name, dnet.CreateOptions{
		Driver: "bridge",
		Labels: netLabels,
	}); err != nil {
		// Tolerate race where another worker created it concurrently.
		if !strings.Contains(err.Error(), "already exists") {
			if isAddressPoolExhaustedError(err) {
				// Best effort cleanup for stale project networks from previous runs.
				_, _ = a.pruneUnusedProjectNetworks(ctx)

				if _, retryErr := a.client.NetworkCreate(ctx, name, dnet.CreateOptions{
					Driver: "bridge",
					Labels: netLabels,
				}); retryErr == nil || strings.Contains(retryErr.Error(), "already exists") {
					return &ports.ProjectScope{
						ProjectID:   projectID,
						ProjectSlug: projectSlug,
						NetworkName: name,
					}, nil
				}

				// Final fallback for local/dev environments where Docker exhausted
				// bridge subnets. This keeps provisioning functional.
				if _, inspectErr := a.client.NetworkInspect(ctx, "bridge", dnet.InspectOptions{}); inspectErr == nil {
					return &ports.ProjectScope{
						ProjectID:   projectID,
						ProjectSlug: projectSlug,
						NetworkName: "bridge",
					}, nil
				}
			}

			return nil, fmt.Errorf("failed to create project network: %w", err)
		}
	}

	return &ports.ProjectScope{
		ProjectID:   projectID,
		ProjectSlug: projectSlug,
		NetworkName: name,
	}, nil
}

func isAddressPoolExhaustedError(err error) bool {
	if err == nil {
		return false
	}
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "all predefined address pools have been fully subnetted") ||
		strings.Contains(lower, "non-overlapping ipv4 address pool")
}

func (a *Adapter) pruneUnusedProjectNetworks(ctx context.Context) (int, error) {
	nets, err := a.client.NetworkList(ctx, dnet.ListOptions{
		Filters: filters.NewArgs(filters.Arg("label", labels.ManagedBy+"="+labels.ManagedByValue)),
	})
	if err != nil {
		return 0, fmt.Errorf("failed to list managed networks: %w", err)
	}

	removed := 0
	for _, n := range nets {
		if !strings.HasPrefix(n.Name, "kleff_proj_") {
			continue
		}

		inspect, inspectErr := a.client.NetworkInspect(ctx, n.ID, dnet.InspectOptions{})
		if inspectErr != nil {
			continue
		}
		if len(inspect.Containers) > 0 {
			continue
		}

		if removeErr := a.client.NetworkRemove(ctx, n.ID); removeErr == nil {
			removed++
		}
	}

	return removed, nil
}

// TeardownProjectScope removes the per-project network. Caller is responsible
// for ensuring no containers remain attached.
func (a *Adapter) TeardownProjectScope(ctx context.Context, projectID string) error {
	if projectID == "" {
		return fmt.Errorf("project id is required")
	}
	nets, err := a.client.NetworkList(ctx, dnet.ListOptions{
		Filters: filters.NewArgs(filters.Arg("label", labels.ProjectID+"="+projectID)),
	})
	if err != nil {
		return fmt.Errorf("failed to list project networks: %w", err)
	}
	for _, n := range nets {
		if err := a.client.NetworkRemove(ctx, n.ID); err != nil {
			return fmt.Errorf("failed to remove project network %s: %w", n.Name, err)
		}
	}
	return nil
}

// Deploy pulls the image and starts a new container.
func (a *Adapter) Deploy(ctx context.Context, spec ports.WorkloadSpec) (*ports.RunningServer, error) {
	if spec.ProjectID == "" {
		return nil, fmt.Errorf("workload spec missing project_id")
	}

	scope, err := a.EnsureProjectScope(ctx, spec.ProjectID, spec.ProjectSlug)
	if err != nil {
		return nil, err
	}

	// Pull image. If the registry is unreachable or denies access, fall back to
	// whatever is already present in the local Docker image cache so that locally
	// built images (e.g. during development) work without a live registry.
	rc, pullErr := a.client.ImagePull(ctx, spec.Image, image.PullOptions{})
	if pullErr != nil {
		if !a.imageExistsLocally(ctx, spec.Image) {
			return nil, fmt.Errorf("failed to pull image %s: %w", spec.Image, pullErr)
		}
		// Image is available locally — proceed without a fresh pull.
	} else {
		if _, err := io.Copy(io.Discard, rc); err != nil {
			rc.Close()
			return nil, fmt.Errorf("failed to pull image %s: %w", spec.Image, err)
		}
		rc.Close()
	}

	containerID, err := a.createContainer(ctx, spec, scope)
	if err != nil {
		return nil, err
	}

	if err := a.client.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	return &ports.RunningServer{
		Labels: labels.WorkloadLabels{
			OwnerID:     spec.OwnerID,
			ServerID:    spec.ServerID,
			BlueprintID: spec.BlueprintID,
			NodeID:      a.nodeID,
			ProjectID:   spec.ProjectID,
			ProjectSlug: spec.ProjectSlug,
		},
		RuntimeRef: containerID,
		State:      "Running",
	}, nil
}

// Start restarts a stopped container. If it no longer exists, re-creates it.
func (a *Adapter) Start(ctx context.Context, spec ports.WorkloadSpec) (*ports.RunningServer, error) {
	if spec.ProjectID == "" {
		return nil, fmt.Errorf("workload spec missing project_id")
	}
	containerID, err := a.findContainer(ctx, spec.ProjectID, spec.ServerID)
	if err != nil {
		if errors.Is(err, ports.ErrProjectMismatch) {
			return nil, err
		}
		if !errors.Is(err, errContainerNotFound) {
			return nil, err
		}
		// Container gone — re-create it.
		return a.Deploy(ctx, spec)
	}

	if err := a.client.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	return &ports.RunningServer{RuntimeRef: containerID, State: "Running"}, nil
}

// Stop stops the container without removing it.
func (a *Adapter) Stop(ctx context.Context, projectID, workloadID string) error {
	containerID, err := a.findContainer(ctx, projectID, workloadID)
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
func (a *Adapter) Remove(ctx context.Context, projectID, workloadID string) error {
	containerID, err := a.findContainer(ctx, projectID, workloadID)
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
func (a *Adapter) Status(ctx context.Context, projectID, workloadID string) (*ports.WorkloadHealth, error) {
	containerID, err := a.findContainer(ctx, projectID, workloadID)
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
func (a *Adapter) Endpoint(ctx context.Context, projectID, workloadID string) (string, error) {
	containerID, err := a.findContainer(ctx, projectID, workloadID)
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
func (a *Adapter) Logs(ctx context.Context, projectID, workloadID string, follow bool) (io.ReadCloser, error) {
	containerID, err := a.findContainer(ctx, projectID, workloadID)
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

func (a *Adapter) createContainer(ctx context.Context, spec ports.WorkloadSpec, scope *ports.ProjectScope) (string, error) {
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

	wl := labels.WorkloadLabels{
		OwnerID:     spec.OwnerID,
		ServerID:    spec.ServerID,
		BlueprintID: spec.BlueprintID,
		NodeID:      a.nodeID,
		ProjectID:   spec.ProjectID,
		ProjectSlug: spec.ProjectSlug,
	}
	containerLabels := wl.ToMap()

	resources := container.Resources{}
	if spec.MemoryBytes > 0 {
		resources.Memory = spec.MemoryBytes
	}
	if spec.CPUMillicores > 0 {
		// Docker uses CPU quota: 1 vCPU = 100000 quota per 100000 period
		resources.CPUQuota = spec.CPUMillicores * 100
		resources.CPUPeriod = 100000
	}

	var mounts []mount.Mount
	if spec.RuntimeHints.PersistentStorage && spec.RuntimeHints.StoragePath != "" {
		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeVolume,
			Source: projectVolumeName(spec.ProjectID, spec.ServerID),
			Target: spec.RuntimeHints.StoragePath,
		})
	}

	netConfig := &dnet.NetworkingConfig{
		EndpointsConfig: map[string]*dnet.EndpointSettings{
			scope.NetworkName: {},
		},
	}

	containerConfig := &container.Config{
		Image:        spec.Image,
		Env:          env,
		ExposedPorts: exposedPorts,
		Labels:       containerLabels,
	}
	if spec.Command != "" {
		containerConfig.Cmd = []string{"sh", "-c", spec.Command}
	}

	resp, err := a.client.ContainerCreate(ctx,
		containerConfig,
		&container.HostConfig{
			PortBindings: portBindings,
			Resources:    resources,
			Mounts:       mounts,
		},
		netConfig,
		nil,
		spec.ServerID,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}
	return resp.ID, nil
}

// imageExistsLocally returns true if the image is already present in the local Docker cache.
func (a *Adapter) imageExistsLocally(ctx context.Context, imageName string) bool {
	images, err := a.client.ImageList(ctx, image.ListOptions{})
	if err != nil {
		return false
	}
	for _, img := range images {
		for _, tag := range img.RepoTags {
			if tag == imageName {
				return true
			}
		}
	}
	return false
}


// findContainer looks up a container by workload id and verifies that its
// project label matches the expected project. Returns ErrProjectMismatch if a
// container with the id exists but belongs to a different project — callers
// must treat that as an authorization failure, not a missing workload.
func (a *Adapter) findContainer(ctx context.Context, projectID, workloadID string) (string, error) {
	if workloadID == "" {
		return "", fmt.Errorf("workload id is required")
	}
	containers, err := a.client.ContainerList(ctx, container.ListOptions{
		All: true,
		Filters: filters.NewArgs(
			filters.Arg("label", labels.WorkloadID+"="+workloadID),
			filters.Arg("label", labels.ManagedBy+"="+labels.ManagedByValue),
		),
	})
	if err != nil {
		return "", fmt.Errorf("failed to list containers: %w", err)
	}
	if len(containers) == 0 {
		// Fall back to the deprecated server_id label during the transition.
		containers, err = a.client.ContainerList(ctx, container.ListOptions{
			All: true,
			Filters: filters.NewArgs(
				filters.Arg("label", labels.ServerID+"="+workloadID),
				filters.Arg("label", labels.ManagedBy+"="+labels.ManagedByValue),
			),
		})
		if err != nil {
			return "", fmt.Errorf("failed to list containers: %w", err)
		}
	}
	if len(containers) == 0 {
		return "", fmt.Errorf("%w for workload %s", errContainerNotFound, workloadID)
	}
	c := containers[0]
	if projectID != "" {
		if got := c.Labels[labels.ProjectID]; got != projectID {
			return "", fmt.Errorf("%w: workload %s belongs to project %q, not %q", ports.ErrProjectMismatch, workloadID, got, projectID)
		}
	}
	return c.ID, nil
}

// projectNetworkName derives a short, deterministic bridge network name from
// the project id. Docker network names are case-sensitive and allow [a-zA-Z0-9_.-].
func projectNetworkName(projectID string) string {
	return "kleff_proj_" + shortID(projectID)
}

// projectVolumeName namespaces persistent volumes by project so a stray
// workload id collision across projects cannot cross-mount data.
func projectVolumeName(projectID, workloadID string) string {
	return fmt.Sprintf("kleff_proj_%s_%s_data", shortID(projectID), workloadID)
}

func shortID(s string) string {
	s = strings.ReplaceAll(s, "-", "")
	if len(s) > 12 {
		s = s[:12]
	}
	return s
}
