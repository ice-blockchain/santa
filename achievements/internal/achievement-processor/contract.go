package achievementprocessor

import (
	"github.com/framey-io/go-tarantool"
	"github.com/ice-blockchain/santa/achievements"
)

type (
	UserID    = string
	BadgeName = string

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
		r  achievements.WriteRepository
	}
)
