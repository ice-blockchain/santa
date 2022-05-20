package achievementprocessor

import (
	"context"
	"github.com/framey-io/go-tarantool"
)

type (
	UserID          = string
	BadgeName       = string
	TaskName        = string
	WriteRepository interface {
		AchieveBadge(ctx context.Context, userID UserID, badgeName BadgeName) error
		AchieveTask(ctx context.Context, userID UserID, taskName TaskName) error
		IncrementUserLevel(ctx context.Context, userID UserID) error
	}
	global struct {
		//nolint:unused // Because it is used by the msgpack library for marshalling/unmarshalling.
		_msgpack struct{} `msgpack:",asArray"`
		Key      string
		Value    uint64 // FIXME: Type?? Scalar may be one of boolean, integer, unsigned, double, number, decimal, string, uuid, varbinary, but I cant find golang mapping in docs.
	}

	badgeSourceProcessor struct {
		db tarantool.Connector
	}
	taskSourceProcessor struct {
		db tarantool.Connector
		r  WriteRepository
	}
)
