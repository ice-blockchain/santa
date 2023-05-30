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
		Group("/v1r").
		GET("/tasks/x/users/:userId", server.RootHandler(s.GetTasks))
}

// GetTasks godoc
//
// @Schemes
// @Description	Returns all the tasks and provided user's progress for each of them.
// @Tags			Tasks
// @Accept			json
// @Produce		json
// @Param			Authorization	header		string	true	"Insert your access token"	default(Bearer <Add access token here>)
// @Param			userId			path		string	true	"the id of the user you need progress for"
// @Success		200				{array}		tasks.Task
// @Failure		400				{object}	server.ErrorResponse	"if validations fail"
// @Failure		401				{object}	server.ErrorResponse	"if not authorized"
// @Failure		403				{object}	server.ErrorResponse	"if not allowed"
// @Failure		422				{object}	server.ErrorResponse	"if syntax fails"
// @Failure		500				{object}	server.ErrorResponse
// @Failure		504				{object}	server.ErrorResponse	"if request times out"
// @Router			/tasks/x/users/{userId} [GET].
func (s *service) GetTasks( //nolint:gocritic // False negative.
	ctx context.Context,
	req *server.Request[GetTasksArg, []*tasks.Task],
) (*server.Response[[]*tasks.Task], *server.Response[server.ErrorResponse]) {
	if req.Data.UserID != req.AuthenticatedUser.UserID {
		return nil, server.Forbidden(errors.Errorf("not allowed. %v != %v", req.Data.UserID, req.AuthenticatedUser.UserID))
	}
	resp, err := s.tasksRepository.GetTasks(ctx, req.Data.UserID)
	if err != nil {
		err = errors.Wrapf(err, "failed to GetTasks for data:%#v", req.Data)

		return nil, server.Unexpected(err)
	}

	return server.OK(&resp), nil
}
