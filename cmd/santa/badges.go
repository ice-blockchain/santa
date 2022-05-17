// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ice-blockchain/wintr/server"
	"github.com/pkg/errors"
)

// GetUserBadges godoc
// @Schemes
// @Description  Returns the badges for an user
// @Tags         Achievements
// @Accept       json
// @Produce      json
// @Param        Authorization  header    string  true  "Insert your access token"  default(Bearer <Add access token here>)
// @Param        userId         path      string  true  "ID of the user"
// @Param        badgeType      query     string  true  "The type of the badges you want. It can be `LEVEL`, `SOCIAL` or `ICE`"
// @Success      200            {array}   achievements.BadgeInventory
// @Failure      400            {object}  server.ErrorResponse  "if validations fail"
// @Failure      401            {object}  server.ErrorResponse  "if not authorized"
// @Failure      404            {object}  server.ErrorResponse  "if user not found"
// @Failure      422            {object}  server.ErrorResponse  "if syntax fails"
// @Failure      500            {object}  server.ErrorResponse
// @Failure      504            {object}  server.ErrorResponse  "if request times out"
// @Router       /user-achievements/{userId}/badges [GET].
func (s *service) GetUserBadges(ctx context.Context, r server.ParsedRequest) server.Response {
	req := r.(*RequestGetUserBadges)
	// User requests its own badges.
	if req.AuthenticatedUser.ID == req.UserID {
		achievedBadges, err := s.achievementsRepository.GetAchievedUserBadges(ctx, req.UserID, req.BadgeType)
		if err != nil {
			return server.Unexpected(err)
		}

		return server.OK(achievedBadges)
	}
	//nolint:nolintlint,godox // TODO not sure if it is valid, need to specify if user can request other user's badges.

	return *server.Forbidden(errors.Errorf("You can request only your own badges"))
}

func newRequestGetUserBadges() server.ParsedRequest {
	return new(RequestGetUserBadges)
}

func (req *RequestGetUserBadges) SetAuthenticatedUser(user server.AuthenticatedUser) {
	if req.AuthenticatedUser.ID == "" {
		req.AuthenticatedUser = user
	}
}

func (req *RequestGetUserBadges) GetAuthenticatedUser() server.AuthenticatedUser {
	return req.AuthenticatedUser
}

func (req *RequestGetUserBadges) Validate() *server.Response {
	b := strings.ToUpper(req.BadgeType)
	if b != badgeTypeLevel && b != badgeTypeSocial && b != badgeTypeIce {
		err := errors.Errorf("badgeType `%v` is not allowed, only one of %v are allowed",
			req.BadgeType, []string{badgeTypeLevel, badgeTypeSocial, badgeTypeIce})

		return &server.Response{
			Code: http.StatusBadRequest,
			Data: server.ErrorResponse{
				Error: err.Error(),
				Code:  "INVALID_PROPERTIES",
			}.Fail(err),
		}
	}

	return server.RequiredStrings(map[string]string{"userId": req.UserID})
}

func (req *RequestGetUserBadges) Bindings(c *gin.Context) []func(obj interface{}) error {
	return []func(obj interface{}) error{c.ShouldBindUri, c.ShouldBindQuery, server.ShouldBindAuthenticatedUser(c)}
}
