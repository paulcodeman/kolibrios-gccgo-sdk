package url

import "strings"

type Error struct {
	Op  string
	URL string
	Err error
}

func (err *Error) Error() string {
	if err == nil {
		return ""
	}
	if err.Err == nil {
		return err.Op + " " + err.URL
	}

	return err.Op + " " + err.URL + ": " + err.Err.Error()
}

func (err *Error) Unwrap() error {
	if err == nil {
		return nil
	}

	return err.Err
}

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

type EscapeError string

func (err EscapeError) Error() string {
	return "invalid URL escape " + quote(string(err))
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

type URL struct {
	Scheme   string
	Opaque   string
	Host     string
	Path     string
	RawPath  string
	RawQuery string
	Fragment string
}

type Values map[string][]string

func Parse(rawURL string) (*URL, error) {
	url := &URL{}
	rest := rawURL

	if fragmentIndex := strings.Index(rest, "#"); fragmentIndex >= 0 {
		url.Fragment = rest[fragmentIndex+1:]
		rest = rest[:fragmentIndex]
	}

	if scheme, remainder, ok := splitScheme(rest); ok {
		url.Scheme = scheme
		rest = remainder
	}

	if url.Scheme != "" && !strings.HasPrefix(rest, "/") && !strings.HasPrefix(rest, "//") {
		queryIndex := strings.Index(rest, "?")
		if queryIndex >= 0 {
			url.Opaque = rest[:queryIndex]
			url.RawQuery = rest[queryIndex+1:]
		} else {
			url.Opaque = rest
		}
		return url, nil
	}

	if strings.HasPrefix(rest, "//") {
		rest = rest[2:]
		authorityEnd := firstIndexAny(rest, "/?")
		if authorityEnd < 0 {
			url.Host = rest
			rest = ""
		} else {
			url.Host = rest[:authorityEnd]
			rest = rest[authorityEnd:]
		}
	}

	queryIndex := strings.Index(rest, "?")
	if queryIndex >= 0 {
		url.Path = rest[:queryIndex]
		url.RawQuery = rest[queryIndex+1:]
	} else {
		url.Path = rest
	}

	return url, nil
}

func (url *URL) String() string {
	if url == nil {
		return ""
	}

	var builder strings.Builder
	if url.Scheme != "" {
		builder.WriteString(url.Scheme)
		builder.WriteByte(':')
	}
	if url.Opaque != "" {
		builder.WriteString(url.Opaque)
	} else {
		if url.Host != "" || strings.HasPrefix(url.Path, "//") {
			builder.WriteString("//")
			builder.WriteString(url.Host)
		}
		builder.WriteString(url.Path)
	}
	if url.RawQuery != "" {
		builder.WriteByte('?')
		builder.WriteString(url.RawQuery)
	}
	if url.Fragment != "" {
		builder.WriteByte('#')
		builder.WriteString(url.Fragment)
	}
	return builder.String()
}

func (url *URL) EscapedPath() string {
	if url == nil {
		return ""
	}
	if url.RawPath != "" {
		return url.RawPath
	}

	return escape(url.Path, escapeModeEscapedPath)
}

func (url *URL) Query() Values {
	if url == nil {
		return make(Values)
	}

	values, _ := ParseQuery(url.RawQuery)
	return values
}

func QueryEscape(value string) string {
	return escape(value, escapeModeQuery)
}

func PathEscape(value string) string {
	return escape(value, escapeModePath)
}

func QueryUnescape(value string) (string, error) {
	return unescape(value, true)
}

func PathUnescape(value string) (string, error) {
	return unescape(value, false)
}

func ParseQuery(query string) (Values, error) {
	values := make(Values)
	if query == "" {
		return values, nil
	}

	parts := strings.Split(query, "&")
	for index := 0; index < len(parts); index++ {
		part := parts[index]
		if part == "" {
			continue
		}

		key := part
		value := ""
		if separator := strings.Index(part, "="); separator >= 0 {
			key = part[:separator]
			value = part[separator+1:]
		}

		unescapedKey, err := QueryUnescape(key)
		if err != nil {
			return values, err
		}
		unescapedValue, err := QueryUnescape(value)
		if err != nil {
			return values, err
		}
		values.Add(unescapedKey, unescapedValue)
	}

	return values, nil
}

func (values Values) Get(key string) string {
	items := values[key]
	if len(items) == 0 {
		return ""
	}

	return items[0]
}

func (values Values) Has(key string) bool {
	_, ok := values[key]
	return ok
}

func (values Values) Set(key string, value string) {
	values[key] = []string{value}
}

func (values Values) Add(key string, value string) {
	values[key] = append(values[key], value)
}

func (values Values) Del(key string) {
	delete(values, key)
}

func (values Values) Encode() string {
	if len(values) == 0 {
		return ""
	}

	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sortStrings(keys)

	var builder strings.Builder
	wrote := false
	for index := 0; index < len(keys); index++ {
		key := keys[index]
		items := values[key]
		for itemIndex := 0; itemIndex < len(items); itemIndex++ {
			if wrote {
				builder.WriteByte('&')
			}
			builder.WriteString(QueryEscape(key))
			builder.WriteByte('=')
			builder.WriteString(QueryEscape(items[itemIndex]))
			wrote = true
		}
	}
	return builder.String()
}

type escapeMode int

const (
	escapeModePath escapeMode = iota
	escapeModeEscapedPath
	escapeModeQuery
)

func escape(value string, mode escapeMode) string {
	if value == "" {
		return ""
	}

	var builder strings.Builder
	for index := 0; index < len(value); index++ {
		current := value[index]
		if current == ' ' && mode == escapeModeQuery {
			builder.WriteByte('+')
			continue
		}
		if shouldNotEscape(current, mode) {
			builder.WriteByte(current)
			continue
		}

		builder.WriteByte('%')
		builder.WriteByte(upperHexDigit(current >> 4))
		builder.WriteByte(upperHexDigit(current & 0x0F))
	}

	return builder.String()
}

func unescape(value string, plusAsSpace bool) (string, error) {
	if value == "" {
		return "", nil
	}

	var builder strings.Builder
	for index := 0; index < len(value); index++ {
		current := value[index]
		if current == '%' {
			if index+2 >= len(value) || !isHexDigit(value[index+1]) || !isHexDigit(value[index+2]) {
				return "", EscapeError(invalidEscapeFragment(value, index))
			}

			builder.WriteByte(decodeHexPair(value[index+1], value[index+2]))
			index += 2
			continue
		}
		if current == '+' && plusAsSpace {
			builder.WriteByte(' ')
			continue
		}

		builder.WriteByte(current)
	}

	return builder.String(), nil
}

func splitScheme(value string) (scheme string, rest string, ok bool) {
	separator := strings.Index(value, ":")
	if separator <= 0 {
		return "", value, false
	}
	if !isValidScheme(value[:separator]) {
		return "", value, false
	}

	return value[:separator], value[separator+1:], true
}

func isValidScheme(value string) bool {
	if value == "" {
		return false
	}
	if !isAlpha(value[0]) {
		return false
	}

	for index := 1; index < len(value); index++ {
		current := value[index]
		if isAlpha(current) || isDigit(current) {
			continue
		}
		if current == '+' || current == '-' || current == '.' {
			continue
		}
		return false
	}

	return true
}

func firstIndexAny(value string, chars string) int {
	best := -1
	for index := 0; index < len(chars); index++ {
		current := strings.Index(value, chars[index:index+1])
		if current < 0 {
			continue
		}
		if best < 0 || current < best {
			best = current
		}
	}

	return best
}

func shouldNotEscape(value byte, mode escapeMode) bool {
	if isAlpha(value) ||
		isDigit(value) ||
		value == '-' ||
		value == '_' ||
		value == '.' ||
		value == '~' {
		return true
	}

	if mode == escapeModeEscapedPath && value == '/' {
		return true
	}

	return false
}

func upperHexDigit(value byte) byte {
	if value < 10 {
		return '0' + value
	}

	return 'A' + (value - 10)
}

func isHexDigit(value byte) bool {
	return isDigit(value) ||
		(value >= 'a' && value <= 'f') ||
		(value >= 'A' && value <= 'F')
}

func decodeHexPair(high byte, low byte) byte {
	return hexValue(high)<<4 | hexValue(low)
}

func hexValue(value byte) byte {
	switch {
	case value >= '0' && value <= '9':
		return value - '0'
	case value >= 'a' && value <= 'f':
		return value - 'a' + 10
	default:
		return value - 'A' + 10
	}
}

func invalidEscapeFragment(value string, index int) string {
	end := index + 3
	if end > len(value) {
		end = len(value)
	}

	return value[index:end]
}

func quote(value string) string {
	return `"` + value + `"`
}

func isAlpha(value byte) bool {
	return value >= 'A' && value <= 'Z' || value >= 'a' && value <= 'z'
}

func isDigit(value byte) bool {
	return value >= '0' && value <= '9'
}

func sortStrings(values []string) {
	for index := 1; index < len(values); index++ {
		current := values[index]
		position := index - 1
		for position >= 0 && stringLess(current, values[position]) {
			values[position+1] = values[position]
			position--
		}
		values[position+1] = current
	}
}

func stringLess(left string, right string) bool {
	limit := len(left)
	if len(right) < limit {
		limit = len(right)
	}

	for index := 0; index < limit; index++ {
		if left[index] == right[index] {
			continue
		}
		return left[index] < right[index]
	}

	return len(left) < len(right)
}
