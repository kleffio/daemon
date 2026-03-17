package payloads

// ServerOperationPayload is the typed payload that the Central API pushes
// to the Redis queue for server operations (Provision, Start, Stop, Delete, Reconcile).
// It contains the core tenant identity data along with the blueprint requirements.
type ServerOperationPayload struct {
	// Identity / Tenancy
	OwnerID     string `json:"owner_id"`
	CrateID     string `json:"crate_id"`
	BlueprintID string `json:"blueprint_id"`

	// Blueprint details required for provision/start
	Image            string            `json:"image"`
	EnvOverrides     map[string]string `json:"env_overrides,omitempty"`
	MemoryBytes      int64             `json:"memory_bytes,omitempty"`
	CPUMillicores    int64             `json:"cpu_millicores,omitempty"`
	PortRequirements []PortRequirement `json:"port_requirements,omitempty"`
}

// PortRequirement defines what ports a container needs to expose.
type PortRequirement struct {
	TargetPort int    `json:"target_port"`
	Protocol   string `json:"protocol"` // "tcp" or "udp"
}
