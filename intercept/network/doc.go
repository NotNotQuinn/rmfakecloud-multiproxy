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
//     specific purposes, if needed.
//
// The "X-Envoy-Decorator-Operation" header on responses from the official cloud
// gives a programmer facing name to the network request. For example,
// GET /integrations/v1/ has the header set to "ingress GetIntegrations".
// That is why it's struct is called "GetIntegrationsResp".
//
// Unfortunately one cannot reliably check the header to see what request is being made,
// because rmfakecloud doesn't send this header.
package network
