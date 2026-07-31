package order

const (
	adminOrderStatusPending   uint8 = 1
	adminOrderStatusPaid      uint8 = 2
	adminOrderStatusCancelled uint8 = 3
	adminOrderStatusClosed    uint8 = 4
	adminOrderStatusFinished  uint8 = 5
)

func allowedAdminOrderStatusTransition(current, target uint8) bool {
	if current < adminOrderStatusPending || current > adminOrderStatusFinished ||
		target < adminOrderStatusPending || target > adminOrderStatusFinished {
		return false
	}
	if current == target {
		return true
	}
	switch current {
	case adminOrderStatusPending:
		return target == adminOrderStatusPaid ||
			target == adminOrderStatusCancelled ||
			target == adminOrderStatusClosed
	case adminOrderStatusCancelled, adminOrderStatusClosed:
		return target == adminOrderStatusPending
	default:
		return false
	}
}
