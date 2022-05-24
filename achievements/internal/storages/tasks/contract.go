package tasks

import (
	"context"
	"github.com/framey-io/go-tarantool"
	"github.com/ice-blockchain/santa/achievements/internal/storages/progress"
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
		UserID     UserID
		TaskName   string
		AchievedAt uint64
	}
)

// Private API.
type (
	repository struct {
		db                        tarantool.Connector
		mb                        messagebroker.Client
		publishAchievedTasksTopic string
	}

	// | userSource is source processor to achieve tasks based on user messages from message broker
	// Tasks -> #1,#3,#5
	usersSource struct {
		r Repository
		p progress.Repository
	}
	// | economyMiningSource is source processor to achieve tasks based on first user's mining session (Tasks -> #2)
	economyMiningSource struct {
		r Repository
	}

	task struct {
		//nolint:unused // Because it is used by the msgpack library for marshalling/unmarshalling.
		_msgpack struct{} `msgpack:",asArray"`
		// Primary key.
		Name TaskName
		// Index of the task ( they should be done in specific order).
		Index uint64
	}
	// `achievedTask` is an internal type to store user's achieved tasks in database.
	achievedTask struct {
		//nolint:unused // Because it is used by the msgpack library for marshalling/unmarshalling.
		_msgpack   struct{} `msgpack:",asArray"`
		UserID     UserID
		TaskName   string
		AchievedAt uint64
	}
)

const (
	tasksSpace                = "TASKS"
	t1ReferralsToAchieveTask6 = 5
	defaultUserPictureName    = "default-user-image.jpg"
	taskClaimUsername         = "TASK1"
	taskFirstMiningSession    = "TASK2"
	taskUploadProfilePicture  = "TASK3"
	taskGetFiveReferrals      = "TASK6"
)
