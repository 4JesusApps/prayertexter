package service

type constError string

func (err constError) Error() string {
	return string(err)
}

const (
	ErrNoAvailableIntercessors = constError("no available intercessors")
	ErrIntercessorUnavailable  = constError("intercessor unavailable")
	ErrInvalidPhone            = constError("no valid phone numbers found")
)
