package ports

import "github.com/kleffio/kleff-daemon/pkg/labels"

// RunningServer is the result returned by Deploy/Start.
type RunningServer struct {
	Labels     labels.WorkloadLabels
	RuntimeRef string
	State      string
}

// WorkloadSpec is the typed payload for all workload operations.
// It is a superset of the old ServerOperationPayload — all existing fields carry over.
type WorkloadSpec struct {
	// Identity / Tenancy
	OwnerID     string `json:"owner_id"`
	ServerID    string `json:"server_id"`
	BlueprintID string `json:"blueprint_id"`
	ProjectID   string `json:"project_id,omitempty"`

	// Blueprint details required for provision/start
	Image            string            `json:"image"`
	BlueprintVersion string            `json:"blueprint_version,omitempty"`
	EnvOverrides     map[string]string `json:"env_overrides,omitempty"`
	MemoryBytes      int64             `json:"memory_bytes,omitempty"`
	CPUMillicores    int64             `json:"cpu_millicores,omitempty"`
	PortRequirements []PortRequirement `json:"port_requirements,omitempty"`
	ConfigFiles      []ConfigFile      `json:"config_files,omitempty"`

	RuntimeHints RuntimeHints `json:"runtime_hints,omitempty"`
}

// PortRequirement defines what ports a container needs to expose.
type PortRequirement struct {
	TargetPort int    `json:"target_port"`
	Protocol   string `json:"protocol"` // "tcp" or "udp"
}

// ConfigFile is a pre-rendered file to mount into the container.
type ConfigFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Mode    string `json:"mode,omitempty"` // e.g. "0644"
}

// RuntimeHints carries workload-type-specific hints for the RuntimeAdapter.
type RuntimeHints struct {
	// KubernetesStrategy controls how the k8s RuntimeAdapter deploys this workload.
	//   ""            → standard Deployment + ClusterIP Service (default)
	//   "agones"      → Agones GameServer CRD
	//   "statefulset" → StatefulSet (databases)
	KubernetesStrategy string `json:"kubernetes_strategy,omitempty"`
	ExposeUDP          bool   `json:"expose_udp,omitempty"`
	HealthCheckPath    string `json:"health_check_path,omitempty"`
	HealthCheckPort    int    `json:"health_check_port,omitempty"`
}

// WorkloadHealth is the per-workload status reported in heartbeats.
type WorkloadHealth struct {
	WorkloadID    string `json:"workload_id"`
	BlueprintID   string `json:"blueprint_id"`
	State         string `json:"state"` // running, stopped, failed
	CPUMillicores int64  `json:"cpu_millicores"`
	MemoryMB      int64  `json:"memory_mb"`
	// Game server extension (zero-valued for non-game workloads)
	ActivePlayers int `json:"active_players,omitempty"`
	// HTTP service extension
	HTTPStatus int `json:"http_status,omitempty"`
}
