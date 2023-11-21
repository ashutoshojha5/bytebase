//go:build !release

package cmd

import (
	"github.com/ashutoshojha5/bytebase/backend/common"
	server "github.com/ashutoshojha5/bytebase/backend/sql-server"
)

func activeProfile() server.Profile {
	return server.Profile{
		Mode:                common.ReleaseModeDev,
		BackendHost:         flags.host,
		BackendPort:         flags.port,
		Version:             version,
		GitCommit:           gitcommit,
		MetricConnectionKey: "",
		WorkspaceID:         flags.workspaceID,
	}
}
