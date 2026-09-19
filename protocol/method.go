package protocol

type Method string

const (
	Get     Method = "GET"
	Head           = "HEAD"
	Post           = "POST"
	Put            = "PUT"
	Delete         = "DELETE"
	Connect        = "CONNECT"
	Options        = "OPTIONS"
	Trace          = "TRACE"
)
