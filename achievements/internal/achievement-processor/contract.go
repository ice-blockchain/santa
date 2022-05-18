package achievementprocessor

import "github.com/framey-io/go-tarantool"

type (
	UserID = string

	badgeSourceProcessor struct {
		db tarantool.Connector
	}
	taskSourceProcessor struct {
		db tarantool.Connector
	}
	AchievedTaskMessage struct {
		UserID     UserID
		TaskName   string
		TaskIndex  uint64
		AchievedAt uint64
	}
	AchievedBadgeMessage struct {
		// Primary key.
		Name string
		// Type of badge, one of: SOCIAL (based on referrals), ICE (based on coins), LEVEL ( based on user's level).
		BadgeType string
		// Min-max range of the certain value (based on badgeType) to achieve the badge.
		FromInclusive uint64
		ToInclusive   uint64
		// User
		UserID UserID
		// Time when badge was achieved
		AchievedAt uint64
	}
)
