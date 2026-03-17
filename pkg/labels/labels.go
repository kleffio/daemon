package labels

const (
	OwnerID     = "kleff.io/owner_id"
	CrateID     = "kleff.io/crate_id"
	BlueprintID = "kleff.io/blueprint_id"
	NodeID      = "kleff.io/node_id"
	ManagedBy   = "kleff.io/managed_by"

	ManagedByValue = "kleff-daemon"
)

type CrateLabels struct {
	OwnerID     string
	CrateID     string
	BlueprintID string
	NodeID      string
}

func (l *CrateLabels) ToMap() map[string]string {
	return map[string]string{
		OwnerID:     l.OwnerID,
		CrateID:     l.CrateID,
		BlueprintID: l.BlueprintID,
		NodeID:      l.NodeID,
		ManagedBy:   ManagedByValue,
	}
}

func FromMap(m map[string]string) CrateLabels {
	if m[ManagedBy] != ManagedByValue {
		return CrateLabels{}
	}
	return CrateLabels{
		OwnerID:     m[OwnerID],
		CrateID:     m[CrateID],
		BlueprintID: m[BlueprintID],
		NodeID:      m[NodeID],
	}
}
