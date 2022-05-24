package userprocessor

import (
	"github.com/framey-io/go-tarantool"
)

type (
	UserID = string

	userSourceProcessor struct {
		db tarantool.Connector
	}
	global struct {
		//nolint:unused // Because it is used by the msgpack library for marshalling/unmarshalling.
		_msgpack struct{} `msgpack:",asArray"`
		Key      string
		Value    uint64 // FIXME: Type?? Scalar may be one of boolean, integer, unsigned, double, number, decimal, string, uuid, varbinary, but I cant find golang mapping in docs.
	}

	// `userAchievements` is an internal type to store user achievements badges in database (USER_ACHIEVEMENTS space)
	// TODO maybe it'll be better to move it into achievement package's contract, but it must be public in that case.
	userAchievements struct {
		//nolint:unused // Because it is used by the msgpack library for marshalling/unmarshalling.
		_msgpack struct{} `msgpack:",asArray"`
		UserID   UserID
		// User's role, one of PIONEER, ABMASSADOR, ...3rd...
		Role string
		// User's balance (ICE).
		Balance uint64 // FIXME: it is float64/DOUBLE in freezer for now, how to convert them? Or refactor freezer to uint64.
		Level   uint32
		// Count of user's referrals on Tier 1.
		T1Referrals uint64
	}
)
