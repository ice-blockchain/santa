// SPDX-License-Identifier: BUSL-1.1

package tasks

import (
	"context"
	"time"

	"github.com/framey-io/go-tarantool"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/ice-blockchain/wintr/connectors/storage"
)

// Public API.
var (
	errAlreadyAchieved = storage.ErrDuplicate
)

type (
	TaskName = string
	UserID   = string

	Repository interface {
		CompleteTask(context.Context, UserID, TaskName) error
	}

	// | CompletedTask is a message broker notification event when user achieves a new task.
	CompletedTask struct {
		AchievedAt time.Time `json:"achievedAt"`
		UserID     UserID    `json:"userId"`
		TaskName   string    `json:"taskName"`
	}
)

// Private API.
type (
	repository struct {
		db tarantool.Connector
		mb messagebroker.Client
	}

	// | userSource is source processor to achieve tasks based on user messages from message broker (Tasks -> #1,#3).
	usersSource struct {
		r Repository
	}
	// | economyMiningSource is source processor to achieve tasks based on first user's mining session (Tasks -> #2).
	economyMiningSource struct {
		r Repository
	}

	// | progressSource is source processor to achieve tasks based on user progress messages from message broker (Tasks -> #6, T1 referrals).
	progressSource struct {
		r Repository
	}

	config struct {
		MessageBroker struct {
			Topics []struct {
				Name string `yaml:"name" json:"name"`
			} `yaml:"topics"`
		} `yaml:"messageBroker"`

		Tasks struct {
			DefaultUserPictureName string `yaml:"defaultUserPictureName"`
			T1Referrals            uint64 `yaml:"t1Referrals"`
		} `yaml:"tasks"`
	}
)

const (
	taskClaimUsername        = "TASK1"
	taskFirstMiningSession   = "TASK2"
	taskUploadProfilePicture = "TASK3"
	taskGetFiveReferrals     = "TASK6"
)

//nolint:gochecknoglobals // Because its loaded once, at runtime.
var cfg config
