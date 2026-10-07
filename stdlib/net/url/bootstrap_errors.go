package url

// The bootstrap errors.As implementation delegates concrete assignments to As.
func (err *Error) As(target interface{}) bool {
	if err == nil {
		return false
	}
	switch typed := target.(type) {
	case **Error:
		if typed == nil {
			return false
		}
		*typed = err
		return true
	case *error:
		if typed == nil {
			return false
		}
		*typed = err
		return true
	}
	return false
}

func (err EscapeError) As(target interface{}) bool {
	switch typed := target.(type) {
	case *EscapeError:
		if typed == nil {
			return false
		}
		*typed = err
		return true
	case *error:
		if typed == nil {
			return false
		}
		*typed = err
		return true
	}
	return false
}
