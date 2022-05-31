// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"

	"github.com/ice-blockchain/santa/achievements"
	"github.com/ice-blockchain/wintr/server"
)

func (s *service) setupAchievementRoutes(router *gin.Engine) {
	router.
		Group("/v1").
		POST("user-achievements/:userId/tasks", server.RootHandler(newRequestCompleteTask, s.CompleteTask)).
		DELETE("user-achievements/:userId/tasks/:taskName", server.RootHandler(newRequestUnCompleteTask, s.UnCompleteTask))
}

// CompleteTask godoc
// @Schemes
// @Description  Completes a specific task for the user.
// @Tags         Achievements
// @Accept       json
// @Produce      json
// @Param        Authorization  header    string               true  "Insert your access token"  default(Bearer <Add access token here>)
// @Param        userId         path      string               true  "ID of the user"
// @Param        request        body      RequestCompleteTask  true  "Request params"
// @Success      201            {object}  achievements.Task
// @Failure      400            {object}  server.ErrorResponse  "if validations fail"
// @Failure      401            {object}  server.ErrorResponse  "if not authorized"
// @Failure      403            {object}  server.ErrorResponse  "if not allowed"
// @Failure      404            {object}  server.ErrorResponse  "if user not found"
// @Failure      409            {object}  server.ErrorResponse  "if task already completed"
// @Failure      422            {object}  server.ErrorResponse  "if syntax fails"
// @Failure      500            {object}  server.ErrorResponse
// @Failure      504            {object}  server.ErrorResponse  "if request times out"
// @Router       /user-achievements/{userId}/tasks [POST].
func (s *service) CompleteTask(ctx context.Context, r server.ParsedRequest) server.Response {
	t := r.(*RequestCompleteTask).Task
	if err := s.achievementsProcessor.CompleteTask(ctx, &t); err != nil {
		err = errors.Wrapf(err, "could not complete task %v", t)
		if errors.Is(err, achievements.ErrRelationNotFound) {
			return *server.NotFound(err, "USER_NOT_FOUND")
		}
		if errors.Is(err, achievements.ErrNotFound) {
			return *server.BadRequest(err, "INVALID_TASK_NAME")
		}
		if errors.Is(err, achievements.ErrDuplicate) {
			return *server.Conflict(err, "TASK_ALREADY_COMPLETED")
		}

		return server.Unexpected(err)
	}

	return server.Created(t)
}

func newRequestCompleteTask() server.ParsedRequest {
	return new(RequestCompleteTask)
}

func (req *RequestCompleteTask) SetAuthenticatedUser(user server.AuthenticatedUser) {
	if req.AuthenticatedUser.ID == "" {
		req.AuthenticatedUser.ID = user.ID
	}
}

func (req *RequestCompleteTask) GetAuthenticatedUser() server.AuthenticatedUser {
	return req.AuthenticatedUser
}

func (req *RequestCompleteTask) Validate() *server.Response {
	if req.UserID != req.AuthenticatedUser.ID {
		return server.Forbidden(errors.Errorf("operation not allowed. User %v tried too complete task %#v", req.AuthenticatedUser.ID, req.Task))
	}

	return nil
}

func (req *RequestCompleteTask) Bindings(c *gin.Context) []func(obj interface{}) error {
	return []func(obj interface{}) error{c.ShouldBindJSON, c.ShouldBindUri, server.ShouldBindAuthenticatedUser(c)}
}

// UnCompleteTask godoc
// @Schemes
// @Description  Un-Completes a specific task for the user.
// @Tags         Achievements
// @Accept       json
// @Produce      json
// @Param        Authorization  header  string  true  "Insert your access token"  default(Bearer <Add access token here>)
// @Param        userId         path    string  true  "ID of the user"
// @Param        taskName       path    string  true  "The name of the task"
// @Success      200            "ok - uncompleted"
// @Success      204            "already uncompleted"
// @Failure      400            {object}  server.ErrorResponse  "if validations fail"
// @Failure      401            {object}  server.ErrorResponse  "if not authorized"
// @Failure      403            {object}  server.ErrorResponse  "if not allowed"
// @Failure      422            {object}  server.ErrorResponse  "if syntax fails"
// @Failure      500            {object}  server.ErrorResponse
// @Failure      504            {object}  server.ErrorResponse  "if request times out"
// @Router       /user-achievements/{userId}/tasks/{taskName} [DELETE].
func (s *service) UnCompleteTask(ctx context.Context, r server.ParsedRequest) server.Response {
	if err := s.achievementsProcessor.UnCompleteTask(ctx, &r.(*RequestUnCompleteTask).Task); err != nil {
		err = errors.Wrapf(err, "could not uncomplete task %v", r.(*RequestUnCompleteTask).Task)
		if errors.Is(err, achievements.ErrNotFound) {
			return server.NoContent()
		}

		return server.Unexpected(err)
	}

	return server.OK()
}

func newRequestUnCompleteTask() server.ParsedRequest {
	return new(RequestUnCompleteTask)
}

func (req *RequestUnCompleteTask) SetAuthenticatedUser(user server.AuthenticatedUser) {
	if req.AuthenticatedUser.ID == "" {
		req.AuthenticatedUser.ID = user.ID
	}
}

func (req *RequestUnCompleteTask) GetAuthenticatedUser() server.AuthenticatedUser {
	return req.AuthenticatedUser
}

func (req *RequestUnCompleteTask) Validate() *server.Response {
	if req.UserID != req.AuthenticatedUser.ID {
		return server.Forbidden(errors.Errorf("operation not allowed. User %v tried too complete task %#v", req.AuthenticatedUser.ID, req.Task))
	}

	return nil
}

func (req *RequestUnCompleteTask) Bindings(c *gin.Context) []func(obj interface{}) error {
	return []func(obj interface{}) error{c.ShouldBindUri, server.ShouldBindAuthenticatedUser(c)}
}
