//go:build !aws

package service

import (
	"github.com/ashutoshojha5/bytebase/backend/enterprise/plugin"
	"github.com/ashutoshojha5/bytebase/backend/enterprise/plugin/hub"
)

func getLicenseProvider(providerConfig *plugin.ProviderConfig) (plugin.LicenseProvider, error) {
	return hub.NewProvider(providerConfig)
}
