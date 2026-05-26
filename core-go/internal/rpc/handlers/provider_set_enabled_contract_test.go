package handlers

import (
	"testing"
)

func TestContract_ProviderSetEnabled_Response(t *testing.T) {
	schema := loadSchema(t, "methods/provider.setEnabled.json")
	resp := providerSetEnabledResponse{Updated: true}
	validateAgainstSchema(t, schema, resp)
}
