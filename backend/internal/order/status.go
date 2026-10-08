package order

// This file holds the order status state machine (AC-04). The canonical
// workflow order and NextStatus live in model.go; everything here builds on
// them so there is exactly one definition of the flow.

// IsAllowedTransition reports whether target may directly follow current.
//
// Only the next status in the workflow is accepted, so both a skipped step
// (requested -> in_progress) and a step back (done -> in_progress) are
// rejected. An unknown current or target status is never a valid transition.
func IsAllowedTransition(current, target string) bool {
	next, ok := NextStatus(current)
	return ok && next == target
}
