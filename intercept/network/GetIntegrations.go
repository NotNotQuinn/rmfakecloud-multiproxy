package network

import "time"

// Response to GET https://internal.cloud.remarkable.com/integrations/v1/
//
// Network request is made every time(?) the user presses "Menu"
// to open the sidebar. If any integrations are returned, the
// "Integrations >" item appears in the menu. Validated in OS 3.5.2.
type GetIntegrationsResp struct {
	Integrations []Integration `json:"integrations"`
}

type Integration struct {
	// Unique ID for this integration.
	ID string `json:"id"`
	// User's unique ID.
	// For the official cloud, it is in the form
	//
	//     "auth0|" + hex
	//     len(hex) == 24
	UserID string `json:"userID"`
	// Set by the user on creation
	Name string `json:"name"`
	// Datetime this integration was added
	Added time.Time `json:"added"`
	// Provider of this integration
	ProviderID string `json:"provider"`
	// The format of the "issues" field is so far unknown,
	// as it is empty most of the time.
	// You may try to revoke authorization from an official
	// integration and see if something appears here.
	Issues []any `json:"issues"`
}
