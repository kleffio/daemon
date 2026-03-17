package payloads_test

import (
	"encoding/json"
	"reflect"
	"testing"
	"github.com/kleffio/gameserver-daemon/internal/workers/payloads"
)

func TestServerOperationPayload_JSON(t *testing.T) {
	payload := payloads.ServerOperationPayload{
		OwnerID:       "user-123",
		CrateID:       "crate-456",
		BlueprintID:   "blue-789",
		Image:         "itzg/minecraft-server:latest",
		MemoryBytes:   1024 * 1024 * 1024,
		CPUMillicores: 1000,
		EnvOverrides: map[string]string{
			"EULA": "TRUE",
		},
		PortRequirements: []payloads.PortRequirement{
			{TargetPort: 25565, Protocol: "tcp"},
		},
	}

	// Marshal to JSON
	bytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Failed to marshal payload to JSON: %v", err)
	}

	// The serialized bytes should contain the json keys representing the exact structure Redis jobs contain.
	// For testing, let's unmarshal and make sure we get the same thing back.
	var unmarshaled payloads.ServerOperationPayload
	if err := json.Unmarshal(bytes, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal from JSON: %v", err)
	}

	if !reflect.DeepEqual(payload, unmarshaled) {
		t.Errorf("Unmarshaled payload does not match original.\nExpected: %+v\nGot: %+v", payload, unmarshaled)
	}
}
