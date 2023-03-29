// SPDX-License-Identifier: ice License 1.0

package main

import (
	"context"

	"github.com/pkg/errors"

	levelsandroles "github.com/ice-blockchain/santa/levels-and-roles"
	"github.com/ice-blockchain/wintr/server"
)

func (s *service) setupLevelsAndRolesRoutes(router *server.Router) {
	router.
		Group("/v1r").
		GET("/achievement-summaries/levels-and-roles/users/:userId", server.RootHandler(s.GetLevelsAndRolesSummary))
}

// GetLevelsAndRolesSummary godoc
//
// @Schemes
// @Description	Returns user's summary about levels & roles.
// @Tags			Levels & Roles
// @Accept			json
// @Produce		json
// @Param			Authorization	header		string	true	"Insert your access token"	default(Bearer <Add access token here>)
// @Param			userId			path		string	true	"the id of the user you need summary for"
// @Success		200				{object}	levelsandroles.Summary
// @Failure		400				{object}	server.ErrorResponse	"if validations fail"
// @Failure		401				{object}	server.ErrorResponse	"if not authorized"
// @Failure		422				{object}	server.ErrorResponse	"if syntax fails"
// @Failure		500				{object}	server.ErrorResponse
// @Failure		504				{object}	server.ErrorResponse	"if request times out"
// @Router			/achievement-summaries/levels-and-roles/users/{userId} [GET].
func (s *service) GetLevelsAndRolesSummary( //nolint:gocritic // False negative.
	ctx context.Context,
	req *server.Request[GetLevelsAndRolesSummaryArg, levelsandroles.Summary],
) (*server.Response[levelsandroles.Summary], *server.Response[server.ErrorResponse]) {
	resp, err := s.levelsAndRolesRepository.GetSummary(ctx, req.Data.UserID)
	if err != nil {
		err = errors.Wrapf(err, "failed to levelsandroles.GetSummary for data:%#v", req.Data)

		return nil, server.Unexpected(err)
	}

	return server.OK(resp), nil
}
