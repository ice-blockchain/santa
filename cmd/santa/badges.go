// SPDX-License-Identifier: ice License 1.0

package main

import (
	"context"

	"github.com/pkg/errors"

	"github.com/ice-blockchain/santa/badges"
	"github.com/ice-blockchain/wintr/server"
)

func (s *service) setupBadgesRoutes(router *server.Router) {
	router.
		Group("/v1r").
		GET("/badges/:badgeType/users/:userId", server.RootHandler(s.GetBadges)).
		GET("/achievement-summaries/badges/users/:userId", server.RootHandler(s.GetBadgeSummary))
}

// GetBadges godoc
//
// @Schemes
// @Description	Returns all badges of the specific type for the user, with the progress for each of them.
// @Tags			Badges
// @Accept			json
// @Produce		json
// @Param			Authorization	header		string	true	"Insert your access token"	default(Bearer <Add access token here>)
// @Param			userId			path		string	true	"the id of the user you need progress for"
// @Param			badgeType		path		string	true	"the type of the badges"	enums(level,coin,social)
// @Success		200				{array}		badges.Badge
// @Failure		400				{object}	server.ErrorResponse	"if validations fail"
// @Failure		401				{object}	server.ErrorResponse	"if not authorized"
// @Failure		403				{object}	server.ErrorResponse	"if not allowed"
// @Failure		422				{object}	server.ErrorResponse	"if syntax fails"
// @Failure		500				{object}	server.ErrorResponse
// @Failure		504				{object}	server.ErrorResponse	"if request times out"
// @Router			/badges/{badgeType}/users/{userId} [GET].
func (s *service) GetBadges( //nolint:gocritic // False negative.
	ctx context.Context,
	req *server.Request[GetBadgesArg, []*badges.Badge],
) (*server.Response[[]*badges.Badge], *server.Response[server.ErrorResponse]) {
	resp, err := s.badgesRepository.GetBadges(ctx, req.Data.GroupType, req.Data.UserID)
	if err != nil {
		err = errors.Wrapf(err, "failed to GetBadges for data:%#v", req.Data)
		if errors.Is(err, badges.ErrHidden) {
			return nil, server.ForbiddenWithCode(err, badgesHiddenErrorCode)
		}

		return nil, server.Unexpected(err)
	}

	return server.OK(&resp), nil
}

// GetBadgeSummary godoc
//
// @Schemes
// @Description	Returns user's summary about badges.
// @Tags			Badges
// @Accept			json
// @Produce		json
// @Param			Authorization	header		string	true	"Insert your access token"	default(Bearer <Add access token here>)
// @Param			userId			path		string	true	"the id of the user you need summary for"
// @Success		200				{array}		badges.BadgeSummary
// @Failure		400				{object}	server.ErrorResponse	"if validations fail"
// @Failure		401				{object}	server.ErrorResponse	"if not authorized"
// @Failure		403				{object}	server.ErrorResponse	"if not allowed"
// @Failure		422				{object}	server.ErrorResponse	"if syntax fails"
// @Failure		500				{object}	server.ErrorResponse
// @Failure		504				{object}	server.ErrorResponse	"if request times out"
// @Router			/achievement-summaries/badges/users/{userId} [GET].
func (s *service) GetBadgeSummary( //nolint:gocritic // False negative.
	ctx context.Context,
	req *server.Request[GetBadgeSummaryArg, []*badges.BadgeSummary],
) (*server.Response[[]*badges.BadgeSummary], *server.Response[server.ErrorResponse]) {
	resp, err := s.badgesRepository.GetSummary(ctx, req.Data.UserID)
	if err != nil {
		err = errors.Wrapf(err, "failed to badges.GetSummary for data:%#v", req.Data)
		if errors.Is(err, badges.ErrHidden) {
			return nil, server.ForbiddenWithCode(err, badgesHiddenErrorCode)
		}

		return nil, server.Unexpected(err)
	}

	return server.OK(&resp), nil
}
