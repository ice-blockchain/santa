// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ice-blockchain/santa/achievements"
	"github.com/ice-blockchain/wintr/server"
	"github.com/pkg/errors"
)

func (s *service) setupAchievementRoutes(router *gin.Engine) {
	router.
		Group("/v1").
		GET("user-achievements/:userId", server.RootHandler(newRequestGetUserAchievements, s.GetUserAchievements)).
		GET("user-achievements/:userId/badges", server.RootHandler(newRequestGetUserBadges, s.GetUserBadges))
}

// GetUserAchievements godoc
// @Schemes
// @Description  Returns the achievements for an user
// @Tags         Achievements
// @Accept       json
// @Produce      json
// @Param        Authorization        header    string    true   "Insert your access token"  default(Bearer <Add access token here>)
// @Param        userId               path      string    true   "ID of the user"
// @Param        includeCollectibles  query     []string  false  "You can include any of [`TASKS`,`BADGES`]."
// @Success      200                  {object}  achievements.UserAchievements
// @Failure      400                  {object}  server.ErrorResponse  "if validations fail"
// @Failure      401                  {object}  server.ErrorResponse  "if not authorized"
// @Failure      404                  {object}  server.ErrorResponse  "if user not found"
// @Failure      422                  {object}  server.ErrorResponse  "if syntax fails"
// @Failure      500                  {object}  server.ErrorResponse
// @Failure      504                  {object}  server.ErrorResponse  "if request times out"
// @Router       /user-achievements/{userId} [GET].
func (s *service) GetUserAchievements(ctx context.Context, r server.ParsedRequest) server.Response {
	req := r.(*RequestGetUserAchievements)

	result, err := s.achievementsRepository.GetUserAchievements(ctx, req.UserID, req.IncludeCollectibles)
	if err != nil {
		if errors.Is(err, achievements.ErrRelationNotFound) {
			return *server.NotFound(err, userNotFoundCode)
		}

		return server.Unexpected(err)
	}

	return server.OK(result)
}

func newRequestGetUserAchievements() server.ParsedRequest {
	return new(RequestGetUserAchievements)
}

func (req *RequestGetUserAchievements) SetAuthenticatedUser(user server.AuthenticatedUser) {
	if req.AuthenticatedUser.ID == "" {
		req.AuthenticatedUser = user
	}
}

func (req *RequestGetUserAchievements) GetAuthenticatedUser() server.AuthenticatedUser {
	return req.AuthenticatedUser
}

func (req *RequestGetUserAchievements) Validate() *server.Response {
	for _, collectible := range req.IncludeCollectibles {
		c := strings.ToUpper(collectible)
		if c != "TASKS" && c != "BADGES" {
			err := errors.Errorf("element `%v` for includeCollectibles is not allowed, only `TASKS` or `BADGES` are", collectible)

			return &server.Response{
				Code: http.StatusBadRequest,
				Data: server.ErrorResponse{
					Error: err.Error(),
					Code:  "INVALID_PROPERTIES",
				}.Fail(err),
			}
		}
	}

	return server.RequiredStrings(map[string]string{"userId": req.UserID})
}

func (req *RequestGetUserAchievements) Bindings(c *gin.Context) []func(obj interface{}) error {
	return []func(obj interface{}) error{c.ShouldBindUri, c.ShouldBindQuery, server.ShouldBindAuthenticatedUser(c)}
}
