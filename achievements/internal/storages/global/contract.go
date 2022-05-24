package global

import (
	"github.com/framey-io/go-tarantool"
	"github.com/ice-blockchain/santa/achievements/internal/storages/progress"
)

// Private API.
type (
	global struct {
		//nolint:unused // Because it is used by the msgpack library for marshalling/unmarshalling.
		_msgpack struct{} `msgpack:",asArray"`
		Key      string
		// For now we're saving only integer, but scalar may be one of
		// boolean, integer, unsigned, double, number, decimal, string, uuid, varbinary,
		// but I cant find golang mapping in docs (interface{}?).
		Value uint64
	}

	totalBadgesSource struct {
		db tarantool.Connector
	}
	// | totalUsersSource is a source processor to count total users
	totalUsersSource struct {
		db tarantool.Connector
		p  progress.Repository
	}
)
