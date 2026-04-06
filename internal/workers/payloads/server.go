package payloads

import "github.com/kleffio/kleff-daemon/internal/application/ports"

// ServerOperationPayload is deprecated: use ports.WorkloadSpec directly.
// Kept as a type alias so existing worker code compiles without changes during transition.
type ServerOperationPayload = ports.WorkloadSpec

// PortRequirement is deprecated: use ports.PortRequirement directly.
type PortRequirement = ports.PortRequirement
