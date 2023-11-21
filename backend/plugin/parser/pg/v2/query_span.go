package v2

import (
	"context"

	"github.com/pkg/errors"

	"github.com/ashutoshojha5/bytebase/backend/plugin/parser/base"
	storepb "github.com/ashutoshojha5/bytebase/proto/generated-go/store"
)

func init() {
	base.RegisterGetQuerySpan(storepb.Engine_POSTGRES, GetQuerySpan)
	base.RegisterGetQuerySpan(storepb.Engine_REDSHIFT, GetQuerySpan)
	base.RegisterGetQuerySpan(storepb.Engine_RISINGWAVE, GetQuerySpan)
}

// GetQuerySpan returns the query span for the given statement.
func GetQuerySpan(ctx context.Context, statement, database string, getDatabaseMetadata base.GetDatabaseMetadataFunc) (*base.QuerySpan, error) {
	extractor := newQuerySpanExtractor(database, getDatabaseMetadata)

	querySpan, err := extractor.getQuerySpan(ctx, statement)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get query span from statement: %s", statement)
	}
	return querySpan, nil
}
