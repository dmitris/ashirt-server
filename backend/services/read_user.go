// Copyright 2020, Verizon Media
// Licensed under the terms of the MIT. See LICENSE file in project root for terms.

package services

import (
	"context"

	sq "github.com/Masterminds/squirrel"
	"github.com/theparanoids/ashirt-server/backend"
	"github.com/theparanoids/ashirt-server/backend/database"
	"github.com/theparanoids/ashirt-server/backend/dtos"
	"github.com/theparanoids/ashirt-server/backend/models"
	"github.com/theparanoids/ashirt-server/backend/policy"
	"github.com/theparanoids/ashirt-server/backend/server/middleware"
)

// ReadUser retrieves a detailed view of a user. This is separate from the data retriving by listing
// users, or reading another user's profile (when not an admin)
func ReadUser(ctx context.Context, db *database.Connection, userSlug string, supportedAuthSchemes *[]dtos.SupportedAuthScheme) (*dtos.UserOwnView, error) {
	userID, err := SelfOrSlugToUserID(ctx, db, userSlug)
	if err != nil {
		return nil, backend.WrapError("Unable to read user", backend.DatabaseErr(err))
	}

	if err := policy.Require(middleware.Policy(ctx), policy.CanReadDetailedUser{UserID: userID}); err != nil {
		return nil, backend.WrapError("Unwilling to read user", backend.UnauthorizedReadErr(err))
	}

	supportedAuthCodes := make([]string, len(*supportedAuthSchemes))
	for i, scheme := range *supportedAuthSchemes {
		supportedAuthCodes[i] = scheme.SchemeCode
	}

	var user models.User
	var authSchemes []models.AuthSchemeData
	err = db.WithTx(ctx, func(tx *database.Transactable) {
		db.Get(&user, sq.Select("first_name", "last_name", "slug", "email", "admin", "headless").
			From("users").
			Where(sq.Eq{"id": userID}))

		db.Select(&authSchemes, sq.Select("user_key", "auth_scheme", "auth_type", "last_login").
			From("auth_scheme_data").
			Where(sq.Eq{
				"user_id":     userID,
				"auth_scheme": supportedAuthCodes,
			}))
	})
	if err != nil {
		return nil, backend.WrapError("Cannot read user", backend.DatabaseErr(err))
	}

	auths := make([]dtos.AuthenticationInfo, len(authSchemes))
	for i, v := range authSchemes {
		index := getMatchingSchemeIndex(supportedAuthSchemes, v.AuthScheme)

		auths[i] = dtos.AuthenticationInfo{
			UserKey:        v.UserKey,
			AuthSchemeCode: v.AuthScheme,
			AuthSchemeType: v.AuthType,
			AuthLogin:      v.LastLogin,
			AuthDetails:    nil,
		}
		if index > -1 {
			auths[i].AuthDetails = &(*supportedAuthSchemes)[index]
		}
	}

	return &dtos.UserOwnView{
		User: dtos.User{
			Slug:      user.Slug,
			FirstName: user.FirstName,
			LastName:  user.LastName,
		},
		Email:          user.Email,
		Admin:          user.Admin,
		Headless:       user.Headless,
		Authentication: auths,
	}, nil
}
