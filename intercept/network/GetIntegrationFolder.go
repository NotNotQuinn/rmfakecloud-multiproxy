package network

import "time"

// Response to GET https://internal.cloud.remarkable.com/integrations/v1/{Integration.ID}/folders/{FolderID}?folderDepth=N
type GetIntegrationFolderResp Folder
type Folder struct {
	Name       string   `json:"name"`
	SubFolders []Folder `json:"subFolders"`
	Files      []File   `json:"files"`
	FolderID   string   `json:"folderID"`
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
	// Unique ID
	FileID string `json:"fileID"`
	// Same value as FileID
	ID string `json:"id"`
	// Extension without the dot.
	//
	// Note: Set to "unknown" for unsupported file extensions.
	FileExtension string `json:"fileExtension"`
	// Same value as FileExtension
	FileType string `json:"fileType"`
}
