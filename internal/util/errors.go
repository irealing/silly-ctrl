package util

type ErrorNo uint64

const (
	NoError ErrorNo = iota
	AuthError
	UnknownAppError
	SignatureTimeoutError
	HandshakeFailedError
	UnknownCommandError
	BadParamError
	SessionAlreadyExists
	ApplicationOver
	UnknownError
)

func (e ErrorNo) Code() uint64 {
	return uint64(e)
}
func (e ErrorNo) String() string {
	switch e {
	case NoError:
		return "success"
	case AuthError:
		return "auth error"
	case UnknownAppError:
		return "unknown app"
	case SignatureTimeoutError:
		return "Signature timeout"
	case HandshakeFailedError:
		return "handshake failed"
	case UnknownCommandError:
		return "unknown command"
	case SessionAlreadyExists:
		return "session already exists"
	case ApplicationOver:
		return "application over"
	default:
		return "unknown"
	}
}
func (e ErrorNo) Error() string {
	return e.String()
}
