package ports

// Metrics records operational metrics for the agent.
type Metrics interface {
	RecordProvision(blueprintID string, success bool)
	RecordStart(blueprintID string, success bool)
	RecordStop(blueprintID string, success bool)
	RecordDelete(blueprintID string, success bool)
	RecordHeartbeat(nodeID string)
	RecordReconcile(nodeID string, count int)
}
