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

var minecraftServerGVR = schema.GroupVersionResource{
	Group:    "kleff.io",
	Version:  "v1alpha1",
	Resource: "minecraftservers",
}

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

func (k *KubernetesRuntime) Start(ctx context.Context, payload payloads.ServerOperationPayload) (*ports.RunningCrate, error) {
	crateLabels := labels.CrateLabels{
		OwnerID:     payload.OwnerID,
		CrateID:     payload.CrateID,
		BlueprintID: payload.BlueprintID,
		NodeID:      k.nodeID,
	}

	labelMap := crateLabels.ToMap()
	labelInterface := make(map[string]interface{})
	for k, v := range labelMap {
		labelInterface[k] = v
	}

	env := payload.EnvOverrides
	spec := map[string]interface{}{
		"serverName":   payload.CrateID,
		"type":         env["TYPE"],
		"version":      env["VERSION"],
		"maxPlayers":   env["MAX_PLAYERS"],
		"difficulty":   env["DIFFICULTY"],
		"gamemode":     env["MODE"],
		"viewDistance": env["VIEW_DISTANCE"],
		"worldSeed":    env["LEVEL_SEED"],
		"onlineMode":   env["ONLINE_MODE"],
	}

	claim := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "kleff.io/v1alpha1",
			"kind":       "MinecraftServer",
			"metadata": map[string]interface{}{
				"name":      payload.CrateID,
				"namespace": k.namespace,
				"labels":    labelInterface,
			},
			"spec": spec,
		},
	}

	_, err := k.client.Resource(minecraftServerGVR).Namespace(k.namespace).Create(ctx, claim, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinecraftServer claim: %w", err)
	}

	crate, err := k.waitForReady(ctx, payload.CrateID, crateLabels)
	if err != nil {
		return nil, fmt.Errorf("server did not reach ready state: %w", err)
	}

	return crate, nil
}

func (k *KubernetesRuntime) Stop(ctx context.Context, crateID string) error {
	return k.client.Resource(minecraftServerGVR).Namespace(k.namespace).Delete(ctx, crateID, metav1.DeleteOptions{})
}

func (k *KubernetesRuntime) Delete(ctx context.Context, crateID string) error {
	return k.Stop(ctx, crateID)
}

func (k *KubernetesRuntime) GetByID(ctx context.Context, crateID string) (*ports.RunningCrate, error) {
	gs, err := k.client.Resource(gameServerGVR).Namespace(k.namespace).Get(ctx, crateID, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get game server: %w", err)
	}

	state, _, _ := unstructured.NestedString(gs.Object, "status", "state")
	rawLabels, _, _ := unstructured.NestedStringMap(gs.Object, "metadata", "labels")

	return &ports.RunningCrate{
		Labels:     labels.FromMap(rawLabels),
		RuntimeRef: crateID,
		State:      state,
	}, nil
}

func (k *KubernetesRuntime) Reconcile(ctx context.Context, nodeID string) ([]*ports.RunningCrate, error) {
	list, err := k.client.Resource(gameServerGVR).Namespace(k.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s,%s=%s", labels.ManagedBy, labels.ManagedByValue, labels.NodeID, nodeID),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list game servers: %w", err)
	}

	var crates []*ports.RunningCrate
	for _, item := range list.Items {
		state, _, _ := unstructured.NestedString(item.Object, "status", "state")
		rawLabels, _, _ := unstructured.NestedStringMap(item.Object, "metadata", "labels")
		crates = append(crates, &ports.RunningCrate{
			Labels:     labels.FromMap(rawLabels),
			RuntimeRef: item.GetName(),
			State:      state,
		})
	}

	return crates, nil
}

func (k *KubernetesRuntime) Stats(ctx context.Context, crateID string) (*ports.RawStats, error) {
	return &ports.RawStats{}, nil
}

func (k *KubernetesRuntime) waitForReady(ctx context.Context, name string, crateLabels labels.CrateLabels) (*ports.RunningCrate, error) {
	var crate *ports.RunningCrate

	err := wait.PollUntilContextTimeout(ctx, 5*time.Second, 5*time.Minute, true, func(ctx context.Context) (bool, error) {
		gs, err := k.client.Resource(gameServerGVR).Namespace(k.namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return false, nil
		}

		state, _, _ := unstructured.NestedString(gs.Object, "status", "state")
		if state != "Ready" {
			return false, nil
		}

		crate = &ports.RunningCrate{
			Labels:     crateLabels,
			RuntimeRef: name,
			State:      "Ready",
		}
		return true, nil
	})

	if err != nil {
		return nil, err
	}

	return crate, nil
}
