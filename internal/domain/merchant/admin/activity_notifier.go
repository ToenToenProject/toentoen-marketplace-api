package admin

import (
	"toentoen-backend/internal/domain/notification"
)

// NotifyOwnerStaffActivity delegates staff activity alerts to the notification domain service.
var NotifyOwnerStaffActivity = notification.NotifyOwnerStaffActivity
