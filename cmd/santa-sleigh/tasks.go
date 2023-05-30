// SPDX-License-Identifier: ice License 1.0

package main

import (
	"context"

	"github.com/pkg/errors"

	"github.com/ice-blockchain/santa/tasks"
	"github.com/ice-blockchain/wintr/server"
)

func (s *service) setupTasksRoutes(router *server.Router) {
	router.
		Group("/v1w").
		PUT("/tasks/:taskType/users/:userId", server.RootHandler(s.PseudoCompleteTask))
}

// PseudoCompleteTask godoc
//
//	@Schemes
//	@Description	Completes the specific task (identified via task type) for the specified user.
//	@Tags			Tasks
//	@Accept			json
//	@Produce		json
//	@Param			Authorization	header	string					true	"Insert your access token"	default(Bearer <Add access token here>)
//	@Param			taskType		path	string					true	"the type of the task"		enums(claim_username,start_mining,upload_profile_picture,follow_us_on_twitter,join_telegram,invite_friends)
//	@Param			userId			path	string					true	"the id of the user that completed the task"
//	@Param			request			body	CompleteTaskRequestBody	false	"Request params. Set it only if task completion requires additional data."
//	@Success		200				"ok"
//	@Failure		400				{object}	server.ErrorResponse	"if validations fail"
//	@Failure		401				{object}	server.ErrorResponse	"if not authorized"
//	@Failure		403				{object}	server.ErrorResponse	"if not allowed"
//	@Failure		404				{object}	server.ErrorResponse	"if user not found"
//	@Failure		422				{object}	server.ErrorResponse	"if syntax fails"
//	@Failure		500				{object}	server.ErrorResponse
//	@Failure		504				{object}	server.ErrorResponse	"if request times out"
//	@Router			/tasks/{taskType}/users/{userId} [PUT].
func (s *service) PseudoCompleteTask( //nolint:gocritic,gocognit // False negative.
	ctx context.Context,
	req *server.Request[CompleteTaskRequestBody, any],
) (*server.Response[any], *server.Response[server.ErrorResponse]) {
	if err := req.Data.validate(); err != nil {
		return nil, server.UnprocessableEntity(errors.Wrap(err, "validations failed"), invalidPropertiesErrorCode)
	}
	if req.Data.TaskType == tasks.JoinTelegramType && (req.Data.Data == nil || req.Data.Data.TelegramUserHandle == "") {
		return nil, server.UnprocessableEntity(errors.Errorf("`data`.`telegramUserHandle` required"), invalidPropertiesErrorCode)
	}
	if req.Data.TaskType == tasks.FollowUsOnTwitterType && (req.Data.Data == nil || req.Data.Data.TwitterUserHandle == "") {
		return nil, server.UnprocessableEntity(errors.Errorf("`data`.`twitterUserHandle` required"), invalidPropertiesErrorCode)
	}
	task := &tasks.Task{
		Data:   req.Data.Data,
		Type:   req.Data.TaskType,
		UserID: req.AuthenticatedUser.UserID,
	}
	if err := s.tasksProcessor.PseudoCompleteTask(ctx, task); err != nil {
		err = errors.Wrapf(err, "failed to PseudoCompleteTask for %#v, userID:%v", req.Data, req.AuthenticatedUser.UserID)
		switch {
		case errors.Is(err, tasks.ErrRelationNotFound):
			return nil, server.NotFound(err, userNotFoundErrorCode)
		default:
			return nil, server.Unexpected(err)
		}
	}

	return server.OK[any](), nil
}

func (arg *CompleteTaskRequestBody) validate() error {
	for _, taskType := range &tasks.AllTypes {
		if taskType == arg.TaskType {
			return nil
		}
	}

	return errors.Errorf("invalid type `%v`", arg.TaskType)
}
