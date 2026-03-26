package labels_test

import (
	"testing"
	"github.com/kleffio/kleff-daemon/pkg/labels"
)

func TestServerLabels_ToMap(t *testing.T) {
	cl := labels.ServerLabels{
		OwnerID:     "user-123",
		ServerID:    "server-456",
		BlueprintID: "blue-789",
		NodeID:      "node-001",
	}

	m := cl.ToMap()

	if expected := "user-123"; m[labels.OwnerID] != expected {
		t.Errorf("Expected OwnerID %q, got %q", expected, m[labels.OwnerID])
	}
	if expected := "server-456"; m[labels.ServerID] != expected {
		t.Errorf("Expected ServerID %q, got %q", expected, m[labels.ServerID])
	}
	if expected := "blue-789"; m[labels.BlueprintID] != expected {
		t.Errorf("Expected BlueprintID %q, got %q", expected, m[labels.BlueprintID])
	}
	if expected := "node-001"; m[labels.NodeID] != expected {
		t.Errorf("Expected NodeID %q, got %q", expected, m[labels.NodeID])
	}
	if expected := labels.ManagedByValue; m[labels.ManagedBy] != expected {
		t.Errorf("Expected ManagedBy %q, got %q", expected, m[labels.ManagedBy])
	}
}

func TestFromMap_Valid(t *testing.T) {
	m := map[string]string{
		labels.OwnerID:     "user-123",
		labels.ServerID:    "server-456",
		labels.BlueprintID: "blue-789",
		labels.NodeID:      "node-001",
		labels.ManagedBy:   labels.ManagedByValue,
		"some-other-label": "ignored",
	}

	cl := labels.FromMap(m)

	if cl.OwnerID != "user-123" {
		t.Errorf("Expected OwnerID user-123, got %q", cl.OwnerID)
	}
	if cl.ServerID != "server-456" {
		t.Errorf("Expected ServerID server-456, got %q", cl.ServerID)
	}
	if cl.BlueprintID != "blue-789" {
		t.Errorf("Expected BlueprintID blue-789, got %q", cl.BlueprintID)
	}
	if cl.NodeID != "node-001" {
		t.Errorf("Expected NodeID node-001, got %q", cl.NodeID)
	}
}

func TestFromMap_InvalidManagedBy(t *testing.T) {
	m := map[string]string{
		labels.OwnerID:   "user-123",
		labels.ServerID:  "server-456",
		labels.ManagedBy: "some-other-system",
	}

	cl := labels.FromMap(m)

	if cl.OwnerID != "" || cl.ServerID != "" {
		t.Errorf("Expected empty struct for unmanaged container, got: %+v", cl)
	}
}

func TestFromMap_MissingManagedBy(t *testing.T) {
	m := map[string]string{
		labels.OwnerID:  "user-123",
		labels.ServerID: "server-456",
	}

	cl := labels.FromMap(m)

	if cl.OwnerID != "" || cl.ServerID != "" {
		t.Errorf("Expected empty struct for unmanaged container, got: %+v", cl)
	}
}
