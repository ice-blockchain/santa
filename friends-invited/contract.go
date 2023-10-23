// SPDX-License-Identifier: ice License 1.0

package friendsinvited

import (
	"context"
	_ "embed"
	"io"

	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/ice-blockchain/wintr/connectors/storage/v2"
)

type (
	Processor interface {
		io.Closer
		CheckHealth(ctx context.Context) error
	}

	Count struct {
		UserID         string `json:"userId" example:"edfd8c02-75e0-4687-9ac2-1ce4723865c4"`
		FriendsInvited uint64 `json:"friendsInvited" example:"5" db:"invited_count"`
	}
)

// Private API.

const (
	applicationYamlKey = "friends-invited"
)

// .
var (
	//go:embed DDL.sql
	ddl string
)

type (
	repository struct {
		cfg      *config
		shutdown func() error
		db       *storage.DB
		mb       messagebroker.Client
	}

	processor struct {
		*repository
	}

	userTableSource struct {
		*processor
	}
	config struct {
		messagebroker.Config `mapstructure:",squash"` //nolint:tagliatelle // Nope.
	}
)
