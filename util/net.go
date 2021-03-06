package util

func IsTemporaryError(err error) bool {
	if tempErr, ok := err.(interface {
		Temporary() bool
	}); ok {
		return tempErr.Temporary()
	}

	return false
}
