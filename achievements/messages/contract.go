package messages

// Dedicated package to avoid import cycle.
type (
	UserID = string
	// | AchievedTaskMessage is a message broker notification event when user achieves a new task.
	AchievedTaskMessage struct {
		UserID     UserID
		TaskName   string
		TaskIndex  uint64
		AchievedAt uint64
	}

	// | AchievedBadgeMessage is a message broker notification event when user achieves a new badge.
	AchievedBadgeMessage struct {
		// Primary key.
		Name string
		// Type of badge, one of: SOCIAL (based on referrals), ICE (based on coins), LEVEL ( based on user's level).
		BadgeType string
		// User.
		UserID UserID
		// Min-max range of the certain value (based on badgeType) to achieve the badge.
		FromInclusive uint64
		ToInclusive   uint64
		// Time when badge was achieved.
		AchievedAt uint64
	}
)
