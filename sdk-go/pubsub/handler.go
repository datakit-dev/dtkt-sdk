package pubsub

// HandlerFunc processes a single message and returns zero or more output messages.
// Used for transform steps and middleware wrapping, NOT for node handlers.
type HandlerFunc func(msg *Message) ([]*Message, error)

// Middleware wraps a HandlerFunc with additional behavior (retry, throttle, etc.).
type Middleware func(h HandlerFunc) HandlerFunc
