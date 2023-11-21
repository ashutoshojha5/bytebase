package server

import (
	// This includes the first-class database, Postgres.

	// Drivers.
	_ "github.com/ashutoshojha5/bytebase/backend/plugin/db/pg"

	// Parsers.
	_ "github.com/ashutoshojha5/bytebase/backend/plugin/parser/pg"

	// Advisors.
	_ "github.com/ashutoshojha5/bytebase/backend/plugin/advisor/pg"

	// Editors.
	_ "github.com/ashutoshojha5/bytebase/backend/plugin/parser/sql/engine/pg"
)
