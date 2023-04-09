// SPDX-License-Identifier: ice License 1.0

package levelsandroles

import (
	"context"
	_ "embed"
	storagev2 "github.com/ice-blockchain/wintr/connectors/storage/v2"
	"github.com/pkg/errors"
	"io"

	"github.com/ice-blockchain/eskimo/users"
	"github.com/ice-blockchain/go-tarantool-client"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
)

// Public API.

const (
	Level1Type  LevelType = "1"
	Level2Type  LevelType = "2"
	Level3Type  LevelType = "3"
	Level4Type  LevelType = "4"
	Level5Type  LevelType = "5"
	Level6Type  LevelType = "6"
	Level7Type  LevelType = "7"
	Level8Type  LevelType = "8"
	Level9Type  LevelType = "9"
	Level10Type LevelType = "10"
	Level11Type LevelType = "11"
	Level12Type LevelType = "12"
	Level13Type LevelType = "13"
	Level14Type LevelType = "14"
	Level15Type LevelType = "15"
	Level16Type LevelType = "16"
	Level17Type LevelType = "17"
	Level18Type LevelType = "18"
	Level19Type LevelType = "19"
	Level20Type LevelType = "20"
	Level21Type LevelType = "21"
)

const (
	SnowmanRoleType    RoleType = "snowman"
	AmbassadorRoleType RoleType = "ambassador"
)

var (
	//nolint:gochecknoglobals // .
	ErrRaceCondition = errors.New("race condition")
	//nolint:gochecknoglobals // It's just for more descriptive validation messages.
	AllLevelTypes = [21]LevelType{
		Level1Type,
		Level2Type,
		Level3Type,
		Level4Type,
		Level5Type,
		Level6Type,
		Level7Type,
		Level8Type,
		Level9Type,
		Level10Type,
		Level11Type,
		Level12Type,
		Level13Type,
		Level14Type,
		Level15Type,
		Level16Type,
		Level17Type,
		Level18Type,
		Level19Type,
		Level20Type,
		Level21Type,
	}
	//nolint:gochecknoglobals // It's just for more descriptive validation messages.
	AllRoleTypes = [2]RoleType{
		SnowmanRoleType,
		AmbassadorRoleType,
	}
	//nolint:gochecknoglobals // It's just for more descriptive validation messages.
	AllRoleTypesThatCanBeEnabled = [1]RoleType{
		AmbassadorRoleType,
	}
)

type (
	LevelType string
	RoleType  string
	Role      struct {
		Type    RoleType `json:"type" example:"snowman"`
		Enabled bool     `json:"enabled" example:"true"`
	}
	Summary struct {
		Roles []*Role `json:"roles,omitempty"`
		Level uint64  `json:"level,omitempty" example:"11"`
	}
	CompletedLevel struct {
		UserID          string    `json:"userId,omitempty" example:"edfd8c02-75e0-4687-9ac2-1ce4723865c4"`
		Type            LevelType `json:"type,omitempty" example:"1"`
		CompletedLevels uint64    `json:"completedLevels,omitempty" example:"3"`
	}
	EnabledRole struct {
		UserID string   `json:"userId,omitempty" example:"edfd8c02-75e0-4687-9ac2-1ce4723865c4"`
		Type   RoleType `json:"type,omitempty" example:"snowman"`
	}
	ReadRepository interface {
		GetSummary(ctx context.Context, userID string) (*Summary, error)
	}
	WriteRepository interface{} //nolint:revive // .
	Repository      interface {
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
	applicationYamlKey               = "levels-and-roles"
	requestingUserIDCtxValueKey      = "requestingUserIDCtxValueKey"
	agendaPhoneNumberHashesBatchSize = 500
)

// .
var (
	//go:embed DDL.lua
	ddl string
	//go:embed DDL.sql
	ddlV2 string
)

type (
	progress struct {
		_msgpack             struct{}               `msgpack:",asArray"` //nolint:unused,tagliatelle,revive,nosnakecase // To insert we need asArray
		EnabledRoles         *users.Enum[RoleType]  `json:"enabledRoles,omitempty" example:"snowman,ambassador"`
		CompletedLevels      *users.Enum[LevelType] `json:"completedLevels,omitempty" example:"1,2"`
		UserID               string                 `json:"userId,omitempty" example:"edfd8c02-75e0-4687-9ac2-1ce4723865c4"`
		PhoneNumberHash      string                 `json:"phoneNumberHash,omitempty" example:"some hash"`
		MiningStreak         uint64                 `json:"miningStreak,omitempty" example:"3"`
		PingsSent            uint64                 `json:"pingsSent,omitempty" example:"3"`
		AgendaContactsJoined uint64                 `json:"agendaContactsJoined,omitempty" example:"3"`
		FriendsInvited       uint64                 `json:"friendsInvited,omitempty" example:"3"`
		CompletedTasks       uint64                 `json:"completedTasks,omitempty" example:"3"`
		HideLevel            bool                   `json:"hideLevel,omitempty" example:"true"`
		HideRole             bool                   `json:"hideRole,omitempty" example:"true"`
	}
	tryCompleteLevelsCommandSource struct {
		*processor
	}
	userTableSource struct {
		*processor
	}
	miningSessionSource struct {
		*processor
	}
	startedDaysOffSource struct {
		*miningSessionSource
	}
	completedTasksSource struct {
		*processor
	}
	userPingsSource struct {
		*processor
	}
	repository struct {
		cfg      *config
		shutdown func() error
		db       tarantool.Connector
		dbV2     *storagev2.DB
		mb       messagebroker.Client
	}
	processor struct {
		*repository
	}
	config struct {
		MiningStreakMilestones                   map[LevelType]uint64     `yaml:"miningStreakMilestones"`
		PingsSentMilestones                      map[LevelType]uint64     `yaml:"pingsSentMilestones"`
		AgendaContactsJoinedMilestones           map[LevelType]uint64     `yaml:"agendaContactsJoinedMilestones"`
		CompletedTasksMilestones                 map[LevelType]uint64     `yaml:"completedTasksMilestones"`
		messagebroker.Config                     `mapstructure:",squash"` //nolint:tagliatelle // Nope.
		RequiredInvitedFriendsToBecomeAmbassador uint64                   `yaml:"requiredInvitedFriendsToBecomeAmbassador"`
	}
)
