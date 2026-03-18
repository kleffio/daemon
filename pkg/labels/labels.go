package labels

const (
	OwnerID     = "kleff.io/owner_id"
	ServerID    = "kleff.io/server_id"
	BlueprintID = "kleff.io/blueprint_id"
	NodeID      = "kleff.io/node_id"
	ManagedBy   = "kleff.io/managed_by"

	ManagedByValue = "kleff-daemon"
)

type ServerLabels struct {
	OwnerID     string
	ServerID    string
	BlueprintID string
	NodeID      string
}

func (l *ServerLabels) ToMap() map[string]string {
	return map[string]string{
		OwnerID:     l.OwnerID,
		ServerID:    l.ServerID,
		BlueprintID: l.BlueprintID,
		NodeID:      l.NodeID,
		ManagedBy:   ManagedByValue,
	}
}

func FromMap(m map[string]string) ServerLabels {
	if m[ManagedBy] != ManagedByValue {
		return ServerLabels{}
	}
	return ServerLabels{
		OwnerID:     m[OwnerID],
		ServerID:    m[ServerID],
		BlueprintID: m[BlueprintID],
		NodeID:      m[NodeID],
	}
}
