//go:build !minimal

package server

import (
	// Drivers under linux.
	_ "github.com/ashutoshojha5/bytebase/backend/plugin/db/obo"
)
