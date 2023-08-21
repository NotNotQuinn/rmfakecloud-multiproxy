// # Network Types
//
// This package defines types for parsing communications between
// the remote cloud server and the remarkable.
//
// Separate from types defined in `ddvk/rmfakecloud` for a few reasons:
//   - Types are defined under an internal package, and can't be imported
//     outside of the repository.
//   - Changes to the types are likely required and may take a while to be merged.
//   - Custom receiver methods may be added on the types defined here for our
//     specific purposes.
//
// The "X-Envoy-Decorator-Operation" header on responses gives a programmer
// facing name to the network request. For example, /integrations/v1/ has the
// header set to "ingress GetIntegrations".
// That is why it's struct is called "GetIntegrationsResp".
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
	// Unique ID
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

// Response to GET https://internal.cloud.remarkable.com/integrations/v1/{Integration.ID}/folders/{FolderID}?folderDepth=N
type GetIntegrationFolderResp Folder
type Folder struct {
	Name       string    `json:"name"`
	SubFolders *[]Folder `json:"subFolders"`
	Files      *[]File   `json:"files"`
	FolderID   string    `json:"folderID"`
	// Same as FolderID
	ID string `json:"id"`
}

type File struct {
	// Name without .ext
	Name         string    `json:"name"`
	Size         int       `json:"size"`
	LastModified time.Time `json:"dateChanged"`
	// MIME type.
	SourceFileType string `json:"sourceFileType"`
	// MIME type.
	// Empty string means this file cannot be opened on the remarkable.
	ProvidedFileType string `json:"providedFileType,omitempty"`
	// Same as FileExtension
	FileType string `json:"fileType"`
	// Unique ID
	FileID string `json:"fileID"`
	// Same as FileID
	ID string `json:"id"`
	// Note: Set to "unknown" for unsupported file types.
	//
	// Ex: "pdf"
	FileExtension string `json:"fileExtension"`
}

// Response to GET https://internal.cloud.remarkable.com/integrations/v1/{Integration.ID}/files/{FileID}
//
// Content-type must be set correctly.
//
// Seen content-types in this response:
//   - application/pdf
//   - application/zip (for application/epub+zip)
type GetIntegrationFileResp []byte

// Response to POST https://internal.cloud.remarkable.com/integrations/v1/{Integration.ID}/files/{FolderID}?name={file.Name}&fileType={file.FileType}
//
// Post body is file body.
// Called when "Exporting" file to integrations.
//   - Notebooks are exported as pdf (3.5)
//   - Epub are exported as pdf (3.5)
//   - Pdf are exported as pdf (3.5)
type PostIntegrationFileResp struct {
	ID string `json:"id"`
}

// Response to GET https://internal.cloud.remarkable.com/integrations/v1/{Integration.ID}/files/{FileID}/metadata
type GetIntegrationFileMetadataResp struct {
	ID string `json:"id"`
	// filename without extension
	Name string `json:"name"`
	// base64 encoded png 156x220 pixels
	Thumbnail string `json:"thumbnail"`
	// MIME type
	SourceFileType string `json:"sourceFileType"`
	// MIME type
	ProvidedFileType string `json:"providedFileType"`
	// Similar to file ext?
	FileType string `json:"fileType"`
}
