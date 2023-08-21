package network

// Response to GET https://internal.cloud.remarkable.com/integrations/v1/{Integration.ID}/files/{FileID}
//
// Content-type must be set correctly.
//
// Seen content-types in this response:
//   - application/pdf
//   - application/zip (for application/epub+zip)
type GetIntegrationFileResp []byte
