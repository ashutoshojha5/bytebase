//go:build aws

package service

import (
	"github.com/ashutoshojha5/bytebase/backend/enterprise/plugin"
	"github.com/ashutoshojha5/bytebase/backend/enterprise/plugin/aws"
)

func getLicenseProvider(providerConfig *plugin.ProviderConfig) (plugin.LicenseProvider, error) {
	return aws.NewProvider(providerConfig)
}
