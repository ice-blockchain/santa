package tasks

import (
	"context"

	"github.com/framey-io/go-tarantool"
	messagebroker "github.com/ice-blockchain/wintr/connectors/message_broker"
	"github.com/ice-blockchain/wintr/connectors/storage"
)

// Public API.
var (
	ErrAlreadyAchieved = storage.ErrDuplicate
)

type (
	TaskName = string
	UserID   = string

	Repository interface {
		AchieveTask(ctx context.Context, userID UserID, taskName TaskName) error
	}

	Task struct {
		// Primary key.
		Name TaskName
		// Index of the task ( they should be done in specific order).
		Index uint64
	}

	// | AchievedTaskMessage is a message broker notification event when user achieves a new task.
	AchievedTaskMessage struct {
		UserID     UserID `json:"userID"`
		TaskName   string `json:"taskName"`
		AchievedAt uint64 `json:"achievedAt"`
	}
)

// Private API.
type (
	repository struct {
		db                        tarantool.Connector
		mb                        messagebroker.Client
		publishAchievedTasksTopic string
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
		r                         Repository
		t1ReferralsToAchieveTask6 uint64
	}

	config struct {
		MessageBroker struct {
			Topics []struct {
				Name string `yaml:"name" json:"name"`
			} `yaml:"topics"`
		} `yaml:"messageBroker"`

		Tasks struct {
			T1Referrals uint64 `yaml:"T1Referrals"`
		} `yaml:"tasks"`
	}
)

const (
	defaultUserPictureName   = "default-user-image.jpg"
	taskClaimUsername        = "TASK1"
	taskFirstMiningSession   = "TASK2"
	taskUploadProfilePicture = "TASK3"
	taskGetFiveReferrals     = "TASK6"
)

//nolint:gochecknoglobals // Because its loaded once, at runtime.
var cfg config
