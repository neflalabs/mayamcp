package mcpserver

// Host is the smallest domain contract the template tools need. Replace it
// with the target domain when adapting MayaMCP; the transport-agnostic server
// in this package does not depend on any filesystem or UI.
type Host interface {
	Name() string
	Version() string
}
