package errors

type errorString struct {
	text string
}

func (err *errorString) Error() string {
	return err.text
}

type unwrapper interface {
	Unwrap() error
}

type multiUnwrapper interface {
	Unwrap() []error
}

type matcher interface {
	Is(error) bool
}

type aser interface {
	As(interface{}) bool
}

func New(text string) error {
	return &errorString{text: text}
}

func Unwrap(err error) error {
	if err == nil {
		return nil
	}

	var value interface{} = err
	unwrapped, ok := value.(unwrapper)
	if !ok {
		return nil
	}

	return unwrapped.Unwrap()
}

func Join(errs ...error) error {
	count := 0
	for index := 0; index < len(errs); index++ {
		if errs[index] != nil {
			count++
		}
	}
	if count == 0 {
		return nil
	}

	joined := &joinError{
		errs: make([]error, 0, count),
	}
	for index := 0; index < len(errs); index++ {
		if errs[index] != nil {
			joined.errs = append(joined.errs, errs[index])
		}
	}

	return joined
}

func Is(err error, target error) bool {
	if target == nil {
		return err == nil
	}

	return is(err, target)
}

func is(err error, target error) bool {
	for err != nil {
		if err == target {
			return true
		}

		var value interface{} = err
		if matched, ok := value.(matcher); ok && matched.Is(target) {
			return true
		}
		if unwrapped, ok := value.(unwrapper); ok {
			err = unwrapped.Unwrap()
			continue
		}
		if unwrapped, ok := value.(multiUnwrapper); ok {
			return anyIs(unwrapped.Unwrap(), target)
		}

		return false
	}

	return false
}

func anyIs(errs []error, target error) bool {
	for index := 0; index < len(errs); index++ {
		err := errs[index]
		if err != nil && is(err, target) {
			return true
		}
	}

	return false
}

func As(err error, target interface{}) bool {
	if err == nil || target == nil {
		return false
	}

	return as(err, target)
}

func as(err error, target interface{}) bool {
	for err != nil {
		if assignAsTarget(err, target) {
			return true
		}

		var value interface{} = err
		if matched, ok := value.(aser); ok && matched.As(target) {
			return true
		}
		if unwrapped, ok := value.(unwrapper); ok {
			err = unwrapped.Unwrap()
			continue
		}
		if unwrapped, ok := value.(multiUnwrapper); ok {
			return anyAs(unwrapped.Unwrap(), target)
		}

		return false
	}

	return false
}

func anyAs(errs []error, target interface{}) bool {
	for index := 0; index < len(errs); index++ {
		err := errs[index]
		if err != nil && as(err, target) {
			return true
		}
	}

	return false
}

func assignAsTarget(err error, target interface{}) bool {
	switch typed := target.(type) {
	case *error:
		if typed == nil {
			return false
		}
		*typed = err
		return true
	case *interface{}:
		if typed == nil {
			return false
		}
		*typed = err
		return true
	}

	return false
}

type joinError struct {
	errs []error
}

func (err *joinError) Error() string {
	if err == nil || len(err.errs) == 0 {
		return ""
	}
	if len(err.errs) == 1 {
		return err.errs[0].Error()
	}

	text := err.errs[0].Error()
	for index := 1; index < len(err.errs); index++ {
		text += "\n" + err.errs[index].Error()
	}

	return text
}

func (err *joinError) Unwrap() []error {
	if err == nil {
		return nil
	}

	return err.errs
}
