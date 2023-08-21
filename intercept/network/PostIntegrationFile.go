package network

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
