// SPDX-License-Identifier: ice License 1.0

package badges

import (
	"context"
	_ "embed"
	"io"

	"github.com/pkg/errors"

	"github.com/ice-blockchain/eskimo/users"
	"github.com/ice-blockchain/wintr/coin"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	storage "github.com/ice-blockchain/wintr/connectors/storage/v2"
)

// Public API.

const (
	Level1Type   Type = "l1"
	Level2Type   Type = "l2"
	Level3Type   Type = "l3"
	Level4Type   Type = "l4"
	Level5Type   Type = "l5"
	Level6Type   Type = "l6"
	Coin1Type    Type = "c1"
	Coin2Type    Type = "c2"
	Coin3Type    Type = "c3"
	Coin4Type    Type = "c4"
	Coin5Type    Type = "c5"
	Coin6Type    Type = "c6"
	Coin7Type    Type = "c7"
	Coin8Type    Type = "c8"
	Coin9Type    Type = "c9"
	Coin10Type   Type = "c10"
	Social1Type  Type = "s1"
	Social2Type  Type = "s2"
	Social3Type  Type = "s3"
	Social4Type  Type = "s4"
	Social5Type  Type = "s5"
	Social6Type  Type = "s6"
	Social7Type  Type = "s7"
	Social8Type  Type = "s8"
	Social9Type  Type = "s9"
	Social10Type Type = "s10"
)

const (
	LevelGroupType  GroupType = "level"
	CoinGroupType   GroupType = "coin"
	SocialGroupType GroupType = "social"
)

var (
	ErrRelationNotFound = storage.ErrRelationNotFound
	ErrHidden           = errors.New("badges are hidden")
	//nolint:gochecknoglobals // It's just for more descriptive validation messages.
	AllTypes = [26]Type{
		Level1Type,
		Level2Type,
		Level3Type,
		Level4Type,
		Level5Type,
		Level6Type,
		Coin1Type,
		Coin2Type,
		Coin3Type,
		Coin4Type,
		Coin5Type,
		Coin6Type,
		Coin7Type,
		Coin8Type,
		Coin9Type,
		Coin10Type,
		Social1Type,
		Social2Type,
		Social3Type,
		Social4Type,
		Social5Type,
		Social6Type,
		Social7Type,
		Social8Type,
		Social9Type,
		Social10Type,
	}
	//nolint:gochecknoglobals,gomnd // .
	LevelTypeOrder = map[Type]int{
		Level1Type: 0,
		Level2Type: 1,
		Level3Type: 2,
		Level4Type: 3,
		Level5Type: 4,
		Level6Type: 5,
	}
	//nolint:gochecknoglobals,gomnd // .
	CoinTypeOrder = map[Type]int{
		Coin1Type:  0,
		Coin2Type:  1,
		Coin3Type:  2,
		Coin4Type:  3,
		Coin5Type:  4,
		Coin6Type:  5,
		Coin7Type:  6,
		Coin8Type:  7,
		Coin9Type:  8,
		Coin10Type: 9,
	}
	//nolint:gochecknoglobals,gomnd // .
	SocialTypeOrder = map[Type]int{
		Social1Type:  0,
		Social2Type:  1,
		Social3Type:  2,
		Social4Type:  3,
		Social5Type:  4,
		Social6Type:  5,
		Social7Type:  6,
		Social8Type:  7,
		Social9Type:  8,
		Social10Type: 9,
	}
	//nolint:gochecknoglobals,gomnd // .
	AllTypeOrder = map[Type]int{
		Level1Type:   0,
		Level2Type:   1,
		Level3Type:   2,
		Level4Type:   3,
		Level5Type:   4,
		Level6Type:   5,
		Coin1Type:    6,
		Coin2Type:    7,
		Coin3Type:    8,
		Coin4Type:    9,
		Coin5Type:    10,
		Coin6Type:    11,
		Coin7Type:    12,
		Coin8Type:    13,
		Coin9Type:    14,
		Coin10Type:   15,
		Social1Type:  16,
		Social2Type:  17,
		Social3Type:  18,
		Social4Type:  19,
		Social5Type:  20,
		Social6Type:  21,
		Social7Type:  22,
		Social8Type:  23,
		Social9Type:  24,
		Social10Type: 25,
	}
	//nolint:gochecknoglobals // .
	LevelTypeNames = map[Type]string{
		Level1Type: "ice Soldier",
		Level2Type: "Wind Sergeant",
		Level3Type: "Snow Lieutenant",
		Level4Type: "Flake Colonel",
		Level5Type: "Frost General",
		Level6Type: "ice Commander",
	}
	//nolint:gochecknoglobals // .
	CoinTypeNames = map[Type]string{
		Coin1Type:  "Poor George",
		Coin2Type:  "Frozen Purse",
		Coin3Type:  "Arctic Assistant",
		Coin4Type:  "iceCube Swagger",
		Coin5Type:  "Polar Consultant",
		Coin6Type:  "Igloo Broker",
		Coin7Type:  "Snowy Accountant",
		Coin8Type:  "Winter Banker",
		Coin9Type:  "Cold Director",
		Coin10Type: "Richie Rich",
	}
	//nolint:gochecknoglobals // .
	SocialTypeNames = map[Type]string{
		Social1Type:  "ice Breaker",
		Social2Type:  "Trouble Maker",
		Social3Type:  "Snowy Plower",
		Social4Type:  "Arctic Prankster",
		Social5Type:  "Glacial Polly",
		Social6Type:  "Frosty Smacker",
		Social7Type:  "Polar Machine",
		Social8Type:  "North Storm",
		Social9Type:  "Snow Fall",
		Social10Type: "ice Legend",
	}
	//nolint:gochecknoglobals // .
	GroupTypeForEachType = map[Type]GroupType{
		Level1Type:   LevelGroupType,
		Level2Type:   LevelGroupType,
		Level3Type:   LevelGroupType,
		Level4Type:   LevelGroupType,
		Level5Type:   LevelGroupType,
		Level6Type:   LevelGroupType,
		Coin1Type:    CoinGroupType,
		Coin2Type:    CoinGroupType,
		Coin3Type:    CoinGroupType,
		Coin4Type:    CoinGroupType,
		Coin5Type:    CoinGroupType,
		Coin6Type:    CoinGroupType,
		Coin7Type:    CoinGroupType,
		Coin8Type:    CoinGroupType,
		Coin9Type:    CoinGroupType,
		Coin10Type:   CoinGroupType,
		Social1Type:  SocialGroupType,
		Social2Type:  SocialGroupType,
		Social3Type:  SocialGroupType,
		Social4Type:  SocialGroupType,
		Social5Type:  SocialGroupType,
		Social6Type:  SocialGroupType,
		Social7Type:  SocialGroupType,
		Social8Type:  SocialGroupType,
		Social9Type:  SocialGroupType,
		Social10Type: SocialGroupType,
	}
	//nolint:gochecknoglobals // .
	AllNames = map[GroupType]map[Type]string{
		LevelGroupType:  LevelTypeNames,
		CoinGroupType:   CoinTypeNames,
		SocialGroupType: SocialTypeNames,
	}
	//nolint:gochecknoglobals // .
	AllGroups = map[GroupType][]Type{
		LevelGroupType:  append(make([]Type, 0, len(LevelTypeOrder)), Level1Type, Level2Type, Level3Type, Level4Type, Level5Type, Level6Type),                                                             //nolint:lll // .
		CoinGroupType:   append(make([]Type, 0, len(CoinTypeOrder)), Coin1Type, Coin2Type, Coin3Type, Coin4Type, Coin5Type, Coin6Type, Coin7Type, Coin8Type, Coin9Type, Coin10Type),                       //nolint:lll // .
		SocialGroupType: append(make([]Type, 0, len(SocialTypeOrder)), Social1Type, Social2Type, Social3Type, Social4Type, Social5Type, Social6Type, Social7Type, Social8Type, Social9Type, Social10Type), //nolint:lll // .
	}
)

type (
	Type           string
	GroupType      string
	AchievingRange struct {
		FromInclusive uint64 `json:"fromInclusive,omitempty"`
		ToInclusive   uint64 `json:"toInclusive,omitempty"`
	}
	Badge struct {
		Name                        string         `json:"name"`
		Type                        Type           `json:"-"`
		GroupType                   GroupType      `json:"type"` //nolint:tagliatelle // Intended.
		PercentageOfUsersInProgress float64        `json:"percentageOfUsersInProgress"`
		Achieved                    bool           `json:"achieved"`
		AchievingRange              AchievingRange `json:"achievingRange"`
	}
	BadgeSummary struct {
		Name      string    `json:"name"`
		GroupType GroupType `json:"type"` //nolint:tagliatelle // Intended.
		Index     uint64    `json:"index"`
		LastIndex uint64    `json:"lastIndex"`
	}
	AchievedBadge struct {
		UserID         string    `json:"userId" example:"edfd8c02-75e0-4687-9ac2-1ce4723865c4"`
		Type           Type      `json:"type" example:"c1"`
		Name           string    `json:"name" example:"Glacial Polly"`
		GroupType      GroupType `json:"groupType" example:"coin"`
		AchievedBadges uint64    `json:"achievedBadges,omitempty" example:"3"`
	}
	ReadRepository interface {
		GetBadges(ctx context.Context, groupType GroupType, userID string) ([]*Badge, error)
		GetSummary(ctx context.Context, userID string) ([]*BadgeSummary, error)
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
	applicationYamlKey          = "badges"
	requestingUserIDCtxValueKey = "requestingUserIDCtxValueKey"
	percent100                  = 100.0
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
		AchievedBadges  *users.Enum[Type] `json:"achievedBadges,omitempty" example:"c1,l1,l2,c2"`
		Balance         *coin.ICEFlake    `json:"balance,omitempty" example:"1232323232"`
		UserID          string            `json:"userId,omitempty" example:"edfd8c02-75e0-4687-9ac2-1ce4723865c4"`
		FriendsInvited  uint64            `json:"friendsInvited,omitempty" example:"3"`
		CompletedLevels uint64            `json:"completedLevels,omitempty" example:"3"`
		HideBadges      bool              `json:"hideBadges,omitempty" example:"false"`
	}
	statistics struct {
		Type       Type
		GroupType  GroupType
		AchievedBy uint64
	}
	tryAchievedBadgesCommandSource struct {
		*processor
	}
	achievedBadgesSource struct {
		*processor
	}
	userTableSource struct {
		*processor
	}
	completedLevelsSource struct {
		*processor
	}
	balancesTableSource struct {
		*processor
	}
	globalTableSource struct {
		*processor
	}
	repository struct {
		cfg      *config
		shutdown func() error
		db       *storage.DB
		mb       messagebroker.Client
	}
	processor struct {
		*repository
	}
	config struct {
		Milestones           map[Type]AchievingRange  `yaml:"milestones"`
		messagebroker.Config `mapstructure:",squash"` //nolint:tagliatelle // Nope.
	}
)
