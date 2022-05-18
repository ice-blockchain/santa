package achievement_processor

import "github.com/framey-io/go-tarantool"

type (
	badgeSourceProcessor struct {
		db tarantool.Connector
	}
)
