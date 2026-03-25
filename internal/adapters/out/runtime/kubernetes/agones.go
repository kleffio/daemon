package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/kleffio/gameserver-daemon/internal/application/ports"
	"github.com/kleffio/gameserver-daemon/internal/workers/payloads"
	"github.com/kleffio/gameserver-daemon/pkg/labels"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
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

var xMinecraftServerGVR = schema.GroupVersionResource{
	Group:    "kleff.io",
	Version:  "v1alpha1",
	Resource: "xminecraftservers",
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

func (k *KubernetesRuntime) Provision(ctx context.Context, payload payloads.ServerOperationPayload) (*ports.RunningServer, error) {
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

	env := payload.EnvOverrides
	maxPlayers, _ := strconv.ParseInt(env["MAX_PLAYERS"], 10, 64)
	viewDistance, _ := strconv.ParseInt(env["VIEW_DISTANCE"], 10, 64)
	onlineMode, _ := strconv.ParseBool(env["ONLINE_MODE"])

	spec := map[string]interface{}{
		"serverName":   payload.ServerID,
		"type":         env["TYPE"],
		"version":      env["VERSION"],
		"maxPlayers":   maxPlayers,
		"difficulty":   env["DIFFICULTY"],
		"gamemode":     env["MODE"],
		"viewDistance": viewDistance,
		"worldSeed":    env["LEVEL_SEED"],
		"onlineMode":   onlineMode,
	}

	claim := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "kleff.io/v1alpha1",
			"kind":       "MinecraftServer",
			"metadata": map[string]interface{}{
				"name":      payload.ServerID,
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

	server, err := k.waitForReady(ctx, payload.ServerID, serverLabels)
	if err != nil {
		return nil, fmt.Errorf("server did not reach ready state: %w", err)
	}

	return server, nil
}

func (k *KubernetesRuntime) Start(ctx context.Context, payload payloads.ServerOperationPayload) (*ports.RunningServer, error) {
	serverLabels := labels.ServerLabels{
		OwnerID:     payload.OwnerID,
		ServerID:    payload.ServerID,
		BlueprintID: payload.BlueprintID,
		NodeID:      k.nodeID,
	}

	compositeName, err := k.getCompositeName(ctx, payload.ServerID)
	if err != nil {
		return nil, err
	}

	patch := []byte(`{"metadata":{"annotations":{"crossplane.io/paused":null}}}`)
	_, err = k.client.Resource(xMinecraftServerGVR).Patch(ctx, compositeName, types.MergePatchType, patch, metav1.PatchOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to unpause composite: %w", err)
	}

	// Delete stale GameServer if it exists so Agones doesn't panic on stale state
	_ = k.client.Resource(gameServerGVR).Namespace(k.namespace).Delete(ctx, payload.ServerID, metav1.DeleteOptions{})

	env := payload.EnvOverrides
	memoryQty := resource.NewQuantity(4*1024*1024*1024, resource.BinarySI)
	if payload.MemoryBytes > 0 {
		memoryQty = resource.NewQuantity(payload.MemoryBytes, resource.BinarySI)
	}
	memory := memoryQty.String()

	gs := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "agones.dev/v1",
			"kind":       "GameServer",
			"metadata": map[string]interface{}{
				"name":      payload.ServerID,
				"namespace": k.namespace,
			},
			"spec": map[string]interface{}{
				"container": "minecraft",
				"health": map[string]interface{}{
					"disabled": true,
				},
				"ports": []interface{}{
					map[string]interface{}{
						"name":          "minecraft",
						"portPolicy":    "Dynamic",
						"containerPort": int64(25565),
						"protocol":      "TCP",
					},
				},
				"template": map[string]interface{}{
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name":  "minecraft",
								"image": "itzg/minecraft-server:latest",
								"env": []interface{}{
									map[string]interface{}{"name": "EULA", "value": "TRUE"},
									map[string]interface{}{"name": "TYPE", "value": env["TYPE"]},
									map[string]interface{}{"name": "VERSION", "value": env["VERSION"]},
									map[string]interface{}{"name": "MAX_PLAYERS", "value": env["MAX_PLAYERS"]},
									map[string]interface{}{"name": "DIFFICULTY", "value": env["DIFFICULTY"]},
									map[string]interface{}{"name": "MODE", "value": env["MODE"]},
									map[string]interface{}{"name": "VIEW_DISTANCE", "value": env["VIEW_DISTANCE"]},
									map[string]interface{}{"name": "LEVEL_SEED", "value": env["LEVEL_SEED"]},
									map[string]interface{}{"name": "ONLINE_MODE", "value": env["ONLINE_MODE"]},
								},
								"resources": map[string]interface{}{
									"requests": map[string]interface{}{"memory": memory},
									"limits":   map[string]interface{}{"memory": memory},
								},
								"volumeMounts": []interface{}{
									map[string]interface{}{
										"name":      "world",
										"mountPath": "/data",
									},
								},
							},
						},
						"volumes": []interface{}{
							map[string]interface{}{
								"name": "world",
								"persistentVolumeClaim": map[string]interface{}{
									"claimName": payload.ServerID,
								},
							},
						},
					},
				},
			},
		},
	}

	if b, jerr := json.MarshalIndent(gs.Object, "", "  "); jerr == nil {
		fmt.Printf("[DEBUG] GameServer spec being submitted:\n%s\n", string(b))
	}

	_, err = k.client.Resource(gameServerGVR).Namespace(k.namespace).Create(ctx, gs, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create game server: %w", err)
	}

	server, err := k.waitForReady(ctx, payload.ServerID, serverLabels)
	if err != nil {
		return nil, fmt.Errorf("server did not reach ready state: %w", err)
	}

	return server, nil
}

func (k *KubernetesRuntime) Stop(ctx context.Context, serverID string) error {
	compositeName, err := k.getCompositeName(ctx, serverID)
	if err != nil {
		return err
	}

	patch := []byte(`{"metadata":{"annotations":{"crossplane.io/paused":"true"}}}`)
	_, err = k.client.Resource(xMinecraftServerGVR).Patch(ctx, compositeName, types.MergePatchType, patch, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("failed to pause composite: %w", err)
	}

	if err := k.client.Resource(gameServerGVR).Namespace(k.namespace).Delete(ctx, serverID, metav1.DeleteOptions{}); err != nil {
		return fmt.Errorf("failed to delete game server: %w", err)
	}

	return nil
}

func (k *KubernetesRuntime) Delete(ctx context.Context, serverID string) error {
	return k.client.Resource(minecraftServerGVR).Namespace(k.namespace).Delete(ctx, serverID, metav1.DeleteOptions{})
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

func (k *KubernetesRuntime) getCompositeName(ctx context.Context, serverID string) (string, error) {
	claim, err := k.client.Resource(minecraftServerGVR).Namespace(k.namespace).Get(ctx, serverID, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get MinecraftServer claim: %w", err)
	}

	compositeName, _, err := unstructured.NestedString(claim.Object, "spec", "resourceRef", "name")
	if err != nil || compositeName == "" {
		return "", fmt.Errorf("composite name not found on claim %s", serverID)
	}

	return compositeName, nil
}

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
