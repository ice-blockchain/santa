package economy_processor

import "github.com/framey-io/go-tarantool"

type (
	economySourceProcessor struct {
		db tarantool.Connector
	}
)
