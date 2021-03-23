package internal

// ErrorConnectionTimeout is returned if the connection through the proxy
// server was not able to be made before the configured timeout expired.
type ErrorConnectionTimeout error

// ErrorUnsupportedScheme is returned if a scheme other than "http" or
// "https" or "ws" or "wss" is used.
type ErrorUnsupportedScheme error
