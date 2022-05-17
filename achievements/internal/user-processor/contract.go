package user_processor

import "github.com/framey-io/go-tarantool"

type (
	userSourceProcessor struct {
		db tarantool.Connector
	}
)
