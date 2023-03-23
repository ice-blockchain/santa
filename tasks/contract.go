// SPDX-License-Identifier: ice License 1.0

package tasks

import (
	"context"
	_ "embed"
	"io"

	"github.com/ice-blockchain/eskimo/users"
	"github.com/ice-blockchain/go-tarantool-client"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/ice-blockchain/wintr/connectors/storage"
)

// Public API.

const (
	ClaimUsernameType        Type = "claim_username"
	StartMiningType          Type = "start_mining"
	UploadProfilePictureType Type = "upload_profile_picture"
	FollowUsOnTwitterType    Type = "follow_us_on_twitter"
	JoinTelegramType         Type = "join_telegram"
	InviteFriendsType        Type = "invite_friends"
)

var (
	ErrRelationNotFound = storage.ErrRelationNotFound
	//nolint:gochecknoglobals // It's just for more descriptive validation messages.
	AllTypes = [6]Type{
		ClaimUsernameType,
		StartMiningType,
		UploadProfilePictureType,
		FollowUsOnTwitterType,
		JoinTelegramType,
		InviteFriendsType,
	}
	//nolint:gochecknoglobals,gomnd // It's just for more descriptive validation messages.
	TypeOrder = map[Type]int{
		ClaimUsernameType:        0,
		StartMiningType:          1,
		UploadProfilePictureType: 2,
		FollowUsOnTwitterType:    3,
		JoinTelegramType:         4,
		InviteFriendsType:        5,
	}
)

type (
	Type string
	Data struct {
		TwitterUserHandle  string `json:"twitterUserHandle,omitempty" example:"jdoe2"`
		TelegramUserHandle string `json:"telegramUserHandle,omitempty" example:"jdoe1"`
		RequiredQuantity   uint64 `json:"requiredQuantity,omitempty" example:"3"`
	}
	Task struct {
		Data      *Data  `json:"data,omitempty"`
		UserID    string `json:"userId,omitempty" swaggerignore:"true" example:"edfd8c02-75e0-4687-9ac2-1ce4723865c4"`
		Type      Type   `json:"type" example:"claim_username"`
		Completed bool   `json:"completed" example:"false"`
	}
	CompletedTask struct {
		UserID         string `json:"userId" example:"edfd8c02-75e0-4687-9ac2-1ce4723865c4"`
		Type           Type   `json:"type" example:"claim_username"`
		CompletedTasks uint64 `json:"completedTasks,omitempty" example:"3"`
	}
	ReadRepository interface {
		GetTasks(ctx context.Context, userID string) ([]*Task, error)
	}
	WriteRepository interface {
		PseudoCompleteTask(ctx context.Context, task *Task) error
	}
	Repository interface {
		io.Closer

		ReadRepository
		WriteRepository
	}
	Processor interface {
		Repository
		CheckHealth(context.Context) error
	}
)

// Private API.

const (
	applicationYamlKey                          = "tasks"
	allTasksCompletionBaseMiningRatePrizeFactor = 150
)

// .
var (
	//go:embed DDL.lua
	ddl string
)

type (
	progress struct {
		_msgpack             struct{}          `msgpack:",asArray"` //nolint:unused,tagliatelle,revive,nosnakecase // To insert we need asArray
		CompletedTasks       *users.Enum[Type] `json:"completedTasks,omitempty" example:"claim_username,start_mining"`
		PseudoCompletedTasks *users.Enum[Type] `json:"pseudoCompletedTasks,omitempty" example:"claim_username,start_mining"`
		UserID               string            `json:"userId,omitempty" example:"edfd8c02-75e0-4687-9ac2-1ce4723865c4"`
		TwitterUserHandle    string            `json:"twitterUserHandle,omitempty" example:"jdoe2"`
		TelegramUserHandle   string            `json:"telegramUserHandle,omitempty" example:"jdoe1"`
		FriendsInvited       uint64            `json:"friendsInvited,omitempty" example:"3"`
		UsernameSet          bool              `json:"usernameSet,omitempty" example:"true"`
		ProfilePictureSet    bool              `json:"profilePictureSet,omitempty" example:"true"`
		MiningStarted        bool              `json:"miningStarted,omitempty" example:"true"`
	}
	tryCompleteTasksCommandSource struct {
		*processor
	}
	miningSessionSource struct {
		*processor
	}
	userTableSource struct {
		*processor
	}
	repository struct {
		cfg      *config
		shutdown func() error
		db       tarantool.Connector
		mb       messagebroker.Client
	}
	processor struct {
		*repository
	}
	config struct {
		messagebroker.Config   `mapstructure:",squash"` //nolint:tagliatelle // Nope.
		RequiredFriendsInvited uint64                   `yaml:"requiredFriendsInvited"`
	}
)
