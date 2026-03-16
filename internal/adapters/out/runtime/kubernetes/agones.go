package agones

import (
	"context"
	"fmt"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"agones.dev/agones/pkg/client/clientset/versioned"
	"github.com/kleffio/gameserver-daemon/internal/application/ports"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type AgonesRuntime struct {
	client    versioned.Interface
	namespace string
}

func New(kubeconfig, namespace string) (*AgonesRuntime, error) {
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

	client, err := versioned.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create agones client: %w", err)
	}

	return &AgonesRuntime{client: client, namespace: namespace}, nil
}

func (a *AgonesRuntime) Provision(ctx context.Context, name string, p ports.ProvisionPayload) (*ports.ServerRecord, error) {
	gs := &agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: a.namespace,
		},
		Spec: agonesv1.GameServerSpec{
			Container: "minecraft",
			Ports: []agonesv1.GameServerPort{
				{
					Name:          "minecraft",
					PortPolicy:    agonesv1.Dynamic,
					ContainerPort: 25565,
					Protocol:      corev1.ProtocolTCP,
				},
			},
			Health: agonesv1.Health{Disabled: true},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "minecraft",
							Image: "itzg/minecraft-server:latest",
							Env: []corev1.EnvVar{
								{Name: "EULA", Value: "TRUE"},
								{Name: "TYPE", Value: p.Type},
								{Name: "VERSION", Value: p.Version},
								{Name: "MAX_PLAYERS", Value: fmt.Sprintf("%d", p.MaxPlayers)},
								{Name: "DIFFICULTY", Value: p.Difficulty},
								{Name: "MODE", Value: p.Gamemode},
								{Name: "VIEW_DISTANCE", Value: fmt.Sprintf("%d", p.ViewDistance)},
								{Name: "LEVEL_SEED", Value: p.WorldSeed},
								{Name: "ONLINE_MODE", Value: fmt.Sprintf("%t", p.OnlineMode)},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse(p.Memory),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse(p.Memory),
								},
							},
						},
						{
							Name:            "mc-monitor",
							Image:           "saulmaldonado/agones-mc",
							Args:            []string{"monitor"},
							ImagePullPolicy: corev1.PullAlways,
							Env: []corev1.EnvVar{
								{Name: "INITIAL_DELAY", Value: "30s"},
								{Name: "MAX_ATTEMPTS", Value: "10"},
								{Name: "INTERVAL", Value: "10s"},
								{Name: "TIMEOUT", Value: "10s"},
							},
						},
					},
				},
			},
		},
	}

	created, err := a.client.AgonesV1().GameServers(a.namespace).Create(ctx, gs, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create game server: %w", err)
	}

	return &ports.ServerRecord{
		ID:         string(created.UID),
		Name:       created.Name,
		Address:    "",
		Port:       0,
		Status:     "provisioning",
		NodeID:     "",
		Runtime:    "agones",
		RuntimeRef: created.Name,
	}, nil
}
