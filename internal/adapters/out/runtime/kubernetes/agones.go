package kubernetes

import (
	"context"
	"fmt"
	"time"

	"github.com/kleffio/gameserver-daemon/internal/application/ports"
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
}

func New(kubeconfig, namespace string) (*KubernetesRuntime, error) {
	var cfg *rest.Config
	var err error

	if kubeconfig == "" {
		cfg, err = rest.InClusterConfig()
	} else {
		cfg, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to build kubeconfig: %w", err)
	}

	client, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	return &KubernetesRuntime{client: client, namespace: namespace}, nil
}

func (k *KubernetesRuntime) Provision(ctx context.Context, name string, p ports.ProvisionPayload) (*ports.ServerRecord, error) {
	claim := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "kleff.io/v1alpha1",
			"kind":       "MinecraftServer",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": k.namespace,
				"labels": map[string]interface{}{
					"kleff.io/managed_by": "kleff-daemon",
				},
			},
			"spec": map[string]interface{}{
				"serverName":   p.ServerName,
				"type":         p.Type,
				"version":      p.Version,
				"maxPlayers":   int64(p.MaxPlayers),
				"difficulty":   p.Difficulty,
				"gamemode":     p.Gamemode,
				"viewDistance": int64(p.ViewDistance),
				"worldSeed":    p.WorldSeed,
				"onlineMode":   p.OnlineMode,
			},
		},
	}

	_, err := k.client.Resource(minecraftServerGVR).Namespace(k.namespace).Create(ctx, claim, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinecraftServer claim: %w", err)
	}

	record, err := k.waitForReady(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("server did not reach ready state: %w", err)
	}

	return record, nil
}

func (k *KubernetesRuntime) waitForReady(ctx context.Context, name string) (*ports.ServerRecord, error) {
	var record *ports.ServerRecord

	err := wait.PollUntilContextTimeout(ctx, 5*time.Second, 5*time.Minute, true, func(ctx context.Context) (bool, error) {
		gs, err := k.client.Resource(gameServerGVR).Namespace(k.namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return false, nil
		}

		state, _, _ := unstructured.NestedString(gs.Object, "status", "state")
		if state != "Ready" {
			return false, nil
		}

		address, _, _ := unstructured.NestedString(gs.Object, "status", "address")
		gsPorts, _, _ := unstructured.NestedSlice(gs.Object, "status", "ports")
		nodeName, _, _ := unstructured.NestedString(gs.Object, "status", "nodeName")

		port := 0
		if len(gsPorts) > 0 {
			portMap, ok := gsPorts[0].(map[string]interface{})
			if ok {
				if p, ok := portMap["port"].(int64); ok {
					port = int(p)
				}
			}
		}

		record = &ports.ServerRecord{
			ID:         string(gs.GetUID()),
			Name:       name,
			Address:    address,
			Port:       port,
			Status:     "running",
			NodeID:     nodeName,
			Runtime:    "agones",
			RuntimeRef: name,
		}
		return true, nil
	})

	if err != nil {
		return nil, err
	}

	return record, nil
}
