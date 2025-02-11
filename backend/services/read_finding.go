// Copyright 2020, Verizon Media
// Licensed under the terms of the MIT. See LICENSE file in project root for terms.

package services

import (
	"context"

	"github.com/theparanoids/ashirt-server/backend"
	"github.com/theparanoids/ashirt-server/backend/database"
	"github.com/theparanoids/ashirt-server/backend/dtos"
	"github.com/theparanoids/ashirt-server/backend/policy"
	"github.com/theparanoids/ashirt-server/backend/server/middleware"

	sq "github.com/Masterminds/squirrel"
)

type ReadFindingInput struct {
	OperationSlug string
	FindingUUID   string
}

func ReadFinding(ctx context.Context, db *database.Connection, i ReadFindingInput) (*dtos.Finding, error) {
	operation, finding, err := lookupOperationFinding(db, i.OperationSlug, i.FindingUUID)
	if err != nil {
		return nil, backend.WrapError("Unable to read finding", backend.UnauthorizedReadErr(err))
	}

	if err := policy.Require(middleware.Policy(ctx), policy.CanReadOperation{OperationID: operation.ID}); err != nil {
		return nil, backend.WrapError("Unwilling to read finding", backend.UnauthorizedReadErr(err))
	}

	var evidenceIDs []int64

	err = db.Select(&evidenceIDs, sq.Select("evidence_id").
		From("evidence_finding_map").
		Where(sq.Eq{"finding_id": finding.ID}))
	if err != nil {
		return nil, backend.WrapError("Cannot load evidence for finding", backend.DatabaseErr(err))
	}

	_, allTags, err := tagsForEvidenceByID(db, evidenceIDs)
	if err != nil {
		return nil, backend.WrapError("Cannot load tags for evidence", backend.DatabaseErr(err))
	}

	var realCategory = ""
	if finding.CategoryID != nil {
		realCategory, err = getFindingCategory(db, *finding.CategoryID)
		if err != nil {
			return nil, backend.WrapError("Cannot load finding category for finding", backend.DatabaseErr(err))
		}
	}

	return &dtos.Finding{
		UUID:          i.FindingUUID,
		Title:         finding.Title,
		Category:      realCategory,
		Description:   finding.Description,
		NumEvidence:   len(evidenceIDs),
		Tags:          allTags,
		ReadyToReport: finding.ReadyToReport,
		TicketLink:    finding.TicketLink,
	}, nil
}
