package labels

const (
	OwnerID     = "kleff.io/owner_id"
	WorkloadID  = "kleff.io/workload_id"
	ServerID    = "kleff.io/server_id" // Deprecated: use WorkloadID; kept for reconcile during rollout
	BlueprintID = "kleff.io/blueprint_id"
	NodeID      = "kleff.io/node_id"
	ProjectID   = "kleff.io/project_id"
	ProjectSlug = "kleff.io/project_slug"
	ManagedBy   = "kleff.io/managed_by"

	ManagedByValue = "kleff-daemon"
)

type WorkloadLabels struct {
	OwnerID     string
	ServerID    string
	BlueprintID string
	NodeID      string
	ProjectID   string
	ProjectSlug string
}

func (l *WorkloadLabels) ToMap() map[string]string {
	m := map[string]string{
		OwnerID:     l.OwnerID,
		WorkloadID:  l.ServerID, // new key
		ServerID:    l.ServerID, // deprecated alias — kept during transition
		BlueprintID: l.BlueprintID,
		NodeID:      l.NodeID,
		ManagedBy:   ManagedByValue,
	}
	if l.ProjectID != "" {
		m[ProjectID] = l.ProjectID
	}
	if l.ProjectSlug != "" {
		m[ProjectSlug] = l.ProjectSlug
	}
	return m
}

func FromMap(m map[string]string) WorkloadLabels {
	if m[ManagedBy] != ManagedByValue {
		return WorkloadLabels{}
	}
	// Accept containers labeled with either the new workload_id key or the old server_id key.
	workloadID := m[WorkloadID]
	if workloadID == "" {
		workloadID = m[ServerID]
	}
	return WorkloadLabels{
		OwnerID:     m[OwnerID],
		ServerID:    workloadID,
		BlueprintID: m[BlueprintID],
		NodeID:      m[NodeID],
		ProjectID:   m[ProjectID],
		ProjectSlug: m[ProjectSlug],
	}
}
