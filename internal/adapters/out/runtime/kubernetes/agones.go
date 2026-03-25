package kubernetes

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/kleffio/gameserver-daemon/internal/application/ports"
	"github.com/kleffio/gameserver-daemon/internal/workers/payloads"
	"github.com/kleffio/gameserver-daemon/pkg/labels"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// gameServerClaimGVR is the kleff.io custom CRD for generic game server claims.
// An operator watches this CRD and creates an agones.dev/v1 GameServer from it.
var gameServerClaimGVR = schema.GroupVersionResource{
	Group:    "kleff.io",
	Version:  "v1alpha1",
	Resource: "gameservers",
}

// gameServerGVR is the Agones GameServer CRD that the operator creates.
// The daemon polls this to wait for Ready state.
var gameServerGVR = schema.GroupVersionResource{
	Group:    "agones.dev",
	Version:  "v1",
	Resource: "gameservers",
}

type KubernetesRuntime struct {
	client    dynamic.Interface
	namespace string
	nodeID    string
}

func New(kubeconfig, namespace, nodeID string) (*KubernetesRuntime, error) {
	var cfg *rest.Config
	var err error

	if kubeconfig == "" {
		cfg, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to build kubeconfig: %w", err)
		}
	} else if strings.HasPrefix(kubeconfig, "http") {
		cfg = &rest.Config{Host: kubeconfig}
	} else {
		cfg, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to build kubeconfig: %w", err)
		}
	}

	client, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	return &KubernetesRuntime{client: client, namespace: namespace, nodeID: nodeID}, nil
}

// Start creates a kleff.io/v1alpha1 GameServer claim. The cluster operator
// watches this CRD and creates an Agones GameServer from it. No game-specific
// logic lives here — image, env vars, and ports come directly from the payload.
func (k *KubernetesRuntime) Start(ctx context.Context, payload payloads.ServerOperationPayload) (*ports.RunningServer, error) {
	serverLabels := labels.ServerLabels{
		OwnerID:     payload.OwnerID,
		ServerID:    payload.ServerID,
		BlueprintID: payload.BlueprintID,
		NodeID:      k.nodeID,
	}

	labelMap := serverLabels.ToMap()
	labelInterface := make(map[string]interface{})
	for k, v := range labelMap {
		labelInterface[k] = v
	}

	// Build env list from all overrides — no game-specific extraction.
	var envList []interface{}
	for name, value := range payload.EnvOverrides {
		envList = append(envList, map[string]interface{}{
			"name":  name,
			"value": value,
		})
	}

	// Build port list from requirements. Default to 25565/TCP if none specified.
	var portList []interface{}
	for _, pr := range payload.PortRequirements {
		portList = append(portList, map[string]interface{}{
			"containerPort": int64(pr.TargetPort),
			"protocol":      strings.ToUpper(pr.Protocol),
		})
	}
	if len(portList) == 0 {
		portList = []interface{}{
			map[string]interface{}{
				"containerPort": int64(25565),
				"protocol":      "TCP",
			},
		}
	}

	claim := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "kleff.io/v1alpha1",
			"kind":       "GameServer",
			"metadata": map[string]interface{}{
				"name":      payload.ServerID,
				"namespace": k.namespace,
				"labels":    labelInterface,
			},
			"spec": map[string]interface{}{
				"serverName":  payload.ServerID,
				"image":       payload.Image,
				"env":         envList,
				"ports":       portList,
				"memoryLimit": formatMemory(payload.MemoryBytes),
				"cpuLimit":    formatCPU(payload.CPUMillicores),
			},
		},
	}

	_, err := k.client.Resource(gameServerClaimGVR).Namespace(k.namespace).Create(ctx, claim, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create GameServer claim: %w", err)
	}

	server, err := k.waitForReady(ctx, payload.ServerID, serverLabels)
	if err != nil {
		return nil, fmt.Errorf("server did not reach ready state: %w", err)
	}

	return server, nil
}

// Stop deletes the kleff.io/v1alpha1 GameServer claim. The operator cleans up
// the Agones GameServer as part of the CRD deletion.
func (k *KubernetesRuntime) Stop(ctx context.Context, serverID string) error {
	return k.client.Resource(gameServerClaimGVR).Namespace(k.namespace).Delete(ctx, serverID, metav1.DeleteOptions{})
}

func (k *KubernetesRuntime) Delete(ctx context.Context, serverID string) error {
	return k.Stop(ctx, serverID)
}

func (k *KubernetesRuntime) GetByID(ctx context.Context, serverID string) (*ports.RunningServer, error) {
	gs, err := k.client.Resource(gameServerGVR).Namespace(k.namespace).Get(ctx, serverID, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get game server: %w", err)
	}

	state, _, _ := unstructured.NestedString(gs.Object, "status", "state")
	rawLabels, _, _ := unstructured.NestedStringMap(gs.Object, "metadata", "labels")

	return &ports.RunningServer{
		Labels:     labels.FromMap(rawLabels),
		RuntimeRef: serverID,
		State:      state,
	}, nil
}

func (k *KubernetesRuntime) Reconcile(ctx context.Context, nodeID string) ([]*ports.RunningServer, error) {
	list, err := k.client.Resource(gameServerGVR).Namespace(k.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s,%s=%s", labels.ManagedBy, labels.ManagedByValue, labels.NodeID, nodeID),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list game servers: %w", err)
	}

	var servers []*ports.RunningServer
	for _, item := range list.Items {
		state, _, _ := unstructured.NestedString(item.Object, "status", "state")
		rawLabels, _, _ := unstructured.NestedStringMap(item.Object, "metadata", "labels")
		servers = append(servers, &ports.RunningServer{
			Labels:     labels.FromMap(rawLabels),
			RuntimeRef: item.GetName(),
			State:      state,
		})
	}

	return servers, nil
}

func (k *KubernetesRuntime) Stats(ctx context.Context, serverID string) (*ports.RawStats, error) {
	return &ports.RawStats{}, nil
}

// formatMemory converts bytes to a Kubernetes memory string (e.g. 4294967296 → "4Gi").
func formatMemory(bytes int64) string {
	const Gi = int64(1 << 30)
	const Mi = int64(1 << 20)
	if bytes > 0 && bytes%Gi == 0 {
		return fmt.Sprintf("%dGi", bytes/Gi)
	}
	if bytes > 0 && bytes%Mi == 0 {
		return fmt.Sprintf("%dMi", bytes/Mi)
	}
	return fmt.Sprintf("%d", bytes)
}

// formatCPU converts millicores to a Kubernetes CPU string (e.g. 1000 → "1000m").
func formatCPU(milliCores int64) string {
	return fmt.Sprintf("%dm", milliCores)
}

// waitForReady polls the Agones GameServer (created by the operator) until it
// reaches Ready state. The operator is responsible for creating the Agones
// resource from the kleff.io GameServer claim.
func (k *KubernetesRuntime) waitForReady(ctx context.Context, name string, serverLabels labels.ServerLabels) (*ports.RunningServer, error) {
	var server *ports.RunningServer

	err := wait.PollUntilContextTimeout(ctx, 5*time.Second, 5*time.Minute, true, func(ctx context.Context) (bool, error) {
		gs, err := k.client.Resource(gameServerGVR).Namespace(k.namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return false, nil
		}

		state, _, _ := unstructured.NestedString(gs.Object, "status", "state")
		if state != "Ready" {
			return false, nil
		}

		server = &ports.RunningServer{
			Labels:     serverLabels,
			RuntimeRef: name,
			State:      "Ready",
		}
		return true, nil
	})

	if err != nil {
		return nil, err
	}

	return server, nil
}
