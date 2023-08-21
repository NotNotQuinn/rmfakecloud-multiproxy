package network

// Response to GET https://internal.cloud.remarkable.com/integrations/v1/{Integration.ID}/files/{FileID}/metadata
type GetIntegrationFileMetadataResp struct {
	// FileID
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
