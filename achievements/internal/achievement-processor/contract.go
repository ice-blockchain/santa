package achievementprocessor

import "github.com/framey-io/go-tarantool"

type (
	badgeSourceProcessor struct {
		db tarantool.Connector
	}
)
