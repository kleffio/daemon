package ports

import "errors"

// ErrPermanent signals that a job failure is unrecoverable and must not be retried.
// Wrap with fmt.Errorf("...: %w", ErrPermanent).
var ErrPermanent = errors.New("permanent failure")
