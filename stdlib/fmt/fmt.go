package fmt

import (
	"errors"
	"internal/reflectlite"
	"io"
	"os"
)

type Stringer interface {
	String() string
}

// State represents the printer state passed to custom formatters.
// It provides access to the io.Writer interface plus information about
// the flags and options for the operand's format specifier.
type State interface {
	// Write is the function to call to emit formatted output to be printed.
	Write(b []byte) (n int, err error)
	// Width returns the value of the width option and whether it has been set.
	Width() (wid int, ok bool)
	// Precision returns the value of the precision option and whether it has been set.
	Precision() (prec int, ok bool)
	// Flag reports whether the flag c, a character, has been set.
	Flag(c int) bool
}

// Formatter is implemented by any value that has a Format method.
// The implementation controls how State and rune are interpreted.
type Formatter interface {
	Format(f State, verb rune)
}

// ScanState represents the scanner state passed to custom scanners.
type ScanState interface {
	ReadRune() (r rune, size int, err error)
	UnreadRune() error
	SkipSpace()
	Token(skipSpace bool, f func(rune) bool) (token []byte, err error)
	Width() (wid int, ok bool)
}

// Scanner is implemented by any value that has a Scan method.
type Scanner interface {
	Scan(state ScanState, verb rune) error
}

type buffer struct {
	data []byte
}

type formatSpec struct {
	verb  byte
	plus  bool
	sharp bool
	zero  bool
	width int
}

func (buffer *buffer) Write(data []byte) (int, error) {
	buffer.data = append(buffer.data, data...)
	return len(data), nil
}

func (buffer *buffer) String() string {
	return string(buffer.data)
}

func (buffer *buffer) writeString(value string) {
	if value == "" {
		return
	}

	buffer.data = append(buffer.data, value...)
}

func (buffer *buffer) writeByte(value byte) {
	buffer.data = append(buffer.data, value)
}

func writeRendered(writer io.Writer, text string) (n int, err error) {
	if text == "" {
		return 0, nil
	}

	written, err := io.WriteString(writer, text)
	if err != nil {
		return written, err
	}
	if written != len(text) {
		return written, io.ErrShortWrite
	}

	return written, nil
}

func renderPrint(values ...interface{}) string {
	buffer := &buffer{}
	for index := 0; index < len(values); index++ {
		buffer.writeString(formatValue(values[index], 'v'))
	}

	return buffer.String()
}

func renderPrintln(values ...interface{}) string {
	buffer := &buffer{}
	for index := 0; index < len(values); index++ {
		if index > 0 {
			buffer.writeByte(' ')
		}
		buffer.writeString(formatValue(values[index], 'v'))
	}
	buffer.writeByte('\n')

	return buffer.String()
}

func renderPrintf(format string, values ...interface{}) string {
	text, _ := renderFormat(format, false, values...)
	return text
}

func renderErrorf(format string, values ...interface{}) (string, []error) {
	return renderFormat(format, true, values...)
}

func renderFormat(format string, allowWrap bool, values ...interface{}) (string, []error) {
	buffer := &buffer{}
	valueIndex := 0
	textStart := 0
	var wrapped []error

	for index := 0; index < len(format); index++ {
		if format[index] != '%' {
			continue
		}

		buffer.writeString(format[textStart:index])
		textStart = index + 1

		if textStart >= len(format) {
			buffer.writeString("%!(NOVERB)")
			return buffer.String(), wrapped
		}

		spec := formatSpec{}
		for textStart < len(format) {
			switch format[textStart] {
			case '+':
				spec.plus = true
				textStart++
			case '#':
				spec.sharp = true
				textStart++
			case '0':
				spec.zero = true
				textStart++
			default:
				goto parsedFlags
			}
		}

		buffer.writeString("%!(NOVERB)")
		return buffer.String(), wrapped

	parsedFlags:
		for textStart < len(format) && isFormatDigit(format[textStart]) {
			spec.width = spec.width*10 + int(format[textStart]-'0')
			textStart++
		}
		if textStart >= len(format) {
			buffer.writeString("%!(NOVERB)")
			return buffer.String(), wrapped
		}

		verb := format[textStart]
		spec.verb = verb
		textStart++
		if verb == '%' {
			buffer.writeByte('%')
			index = textStart - 1
			continue
		}

		if valueIndex >= len(values) {
			buffer.writeString(missingVerb(verb))
			index = textStart - 1
			continue
		}

		if verb == 'w' {
			buffer.writeString(renderWrapValue(values[valueIndex], allowWrap, &wrapped))
			valueIndex++
			index = textStart - 1
			continue
		}

		buffer.writeString(formatValueSpec(values[valueIndex], spec))
		valueIndex++
		index = textStart - 1
	}

	if textStart < len(format) {
		buffer.writeString(format[textStart:])
	}

	return buffer.String(), wrapped
}

func Sprint(values ...interface{}) string {
	buffer := &buffer{}
	_, _ = Fprint(buffer, values...)
	return buffer.String()
}

func Sprintln(values ...interface{}) string {
	buffer := &buffer{}
	_, _ = Fprintln(buffer, values...)
	return buffer.String()
}

func Sprintf(format string, values ...interface{}) string {
	buffer := &buffer{}
	_, _ = Fprintf(buffer, format, values...)
	return buffer.String()
}

func Fprint(writer io.Writer, values ...interface{}) (n int, err error) {
	return writeRendered(writer, renderPrint(values...))
}

func Fprintln(writer io.Writer, values ...interface{}) (n int, err error) {
	return writeRendered(writer, renderPrintln(values...))
}

func Fprintf(writer io.Writer, format string, values ...interface{}) (n int, err error) {
	return writeRendered(writer, renderPrintf(format, values...))
}

func Print(values ...interface{}) (n int, err error) {
	return Fprint(os.DefaultStdout(), values...)
}

func Println(values ...interface{}) (n int, err error) {
	return Fprintln(os.DefaultStdout(), values...)
}

func Printf(format string, values ...interface{}) (n int, err error) {
	return Fprintf(os.DefaultStdout(), format, values...)
}

func Fscan(reader io.Reader, values ...interface{}) (n int, err error) {
	return scanValues(reader, false, values...)
}

func Fscanln(reader io.Reader, values ...interface{}) (n int, err error) {
	return scanValues(reader, true, values...)
}

func Scan(values ...interface{}) (n int, err error) {
	return Fscan(os.DefaultStdin(), values...)
}

func Scanln(values ...interface{}) (n int, err error) {
	return Fscanln(os.DefaultStdin(), values...)
}

func Errorf(format string, values ...interface{}) error {
	text, wrapped := renderErrorf(format, values...)
	switch len(wrapped) {
	case 0:
		return errors.New(text)
	case 1:
		return &wrapError{msg: text, err: wrapped[0]}
	default:
		return &wrapErrors{msg: text, errs: wrapped}
	}
}

type wrapError struct {
	msg string
	err error
}

func (err *wrapError) Error() string {
	if err == nil {
		return ""
	}

	return err.msg
}

func (err *wrapError) Unwrap() error {
	if err == nil {
		return nil
	}

	return err.err
}

type wrapErrors struct {
	msg  string
	errs []error
}

func (err *wrapErrors) Error() string {
	if err == nil {
		return ""
	}

	return err.msg
}

func (err *wrapErrors) Unwrap() []error {
	if err == nil {
		return nil
	}

	return err.errs
}

var errScanSyntax = errors.New("invalid scan syntax")
var errScanTarget = errors.New("unsupported scan target")
var errScanNewline = errors.New("unexpected newline")
var errScanTrailing = errors.New("expected newline")

type scanReader struct {
	reader     io.Reader
	byteBuffer [1]byte
	pending    byte
	haveByte   bool
	lineEnded  bool
}

func scanValues(reader io.Reader, lineMode bool, values ...interface{}) (n int, err error) {
	scanner := &scanReader{reader: reader}

	for index := 0; index < len(values); index++ {
		token, readErr := scanner.readToken(lineMode)
		if readErr != nil {
			return n, readErr
		}
		if assignErr := scanAssign(values[index], token); assignErr != nil {
			return n, assignErr
		}
		n++
	}

	if lineMode {
		if err = scanner.consumeLineTail(); err != nil {
			return n, err
		}
	}

	return n, nil
}

func (scanner *scanReader) readToken(lineMode bool) (string, error) {
	if lineMode && scanner.lineEnded {
		return "", errScanNewline
	}

	for {
		value, err := scanner.readByte()
		if err != nil {
			return "", err
		}
		if isScanNewline(value) {
			if lineMode {
				scanner.lineEnded = true
				return "", errScanNewline
			}
			continue
		}
		if isScanHorizontalSpace(value) {
			continue
		}

		scanner.unreadByte(value)
		break
	}

	token := &buffer{}
	for {
		value, err := scanner.readByte()
		if err != nil {
			if len(token.data) > 0 {
				return token.String(), nil
			}
			return "", err
		}
		if isScanNewline(value) {
			if lineMode {
				scanner.lineEnded = true
			}
			break
		}
		if isScanHorizontalSpace(value) {
			break
		}

		token.writeByte(value)
	}

	if len(token.data) == 0 {
		return "", io.EOF
	}

	return token.String(), nil
}

func (scanner *scanReader) consumeLineTail() error {
	if scanner.lineEnded {
		return nil
	}

	for {
		value, err := scanner.readByte()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if isScanNewline(value) {
			scanner.lineEnded = true
			return nil
		}
		if isScanHorizontalSpace(value) {
			continue
		}

		return errScanTrailing
	}
}

func (scanner *scanReader) readByte() (byte, error) {
	if scanner.haveByte {
		scanner.haveByte = false
		return scanner.pending, nil
	}

	for {
		read, err := scanner.reader.Read(scanner.byteBuffer[:])
		if read > 0 {
			return scanner.byteBuffer[0], nil
		}
		if err != nil {
			return 0, err
		}
	}
}

func (scanner *scanReader) unreadByte(value byte) {
	scanner.pending = value
	scanner.haveByte = true
}

func isScanHorizontalSpace(value byte) bool {
	switch value {
	case ' ', '\t', '\v', '\f':
		return true
	}

	return false
}

func isScanNewline(value byte) bool {
	return value == '\n' || value == '\r'
}

func scanAssign(target interface{}, token string) error {
	switch typed := target.(type) {
	case *string:
		if typed == nil {
			return errScanTarget
		}
		*typed = token
		return nil
	case *bool:
		if typed == nil {
			return errScanTarget
		}
		value, err := parseBoolToken(token)
		if err != nil {
			return err
		}
		*typed = value
		return nil
	case *int:
		if typed == nil {
			return errScanTarget
		}
		value, err := parseSignedToken(token, intBitSize())
		if err != nil {
			return err
		}
		*typed = int(value)
		return nil
	case *int8:
		if typed == nil {
			return errScanTarget
		}
		value, err := parseSignedToken(token, 8)
		if err != nil {
			return err
		}
		*typed = int8(value)
		return nil
	case *int16:
		if typed == nil {
			return errScanTarget
		}
		value, err := parseSignedToken(token, 16)
		if err != nil {
			return err
		}
		*typed = int16(value)
		return nil
	case *int32:
		if typed == nil {
			return errScanTarget
		}
		value, err := parseSignedToken(token, 32)
		if err != nil {
			return err
		}
		*typed = int32(value)
		return nil
	case *int64:
		if typed == nil {
			return errScanTarget
		}
		value, err := parseSignedToken(token, 64)
		if err != nil {
			return err
		}
		*typed = value
		return nil
	case *uint:
		if typed == nil {
			return errScanTarget
		}
		value, err := parseUnsignedToken(token, uintBitSize())
		if err != nil {
			return err
		}
		*typed = uint(value)
		return nil
	case *uint8:
		if typed == nil {
			return errScanTarget
		}
		value, err := parseUnsignedToken(token, 8)
		if err != nil {
			return err
		}
		*typed = uint8(value)
		return nil
	case *uint16:
		if typed == nil {
			return errScanTarget
		}
		value, err := parseUnsignedToken(token, 16)
		if err != nil {
			return err
		}
		*typed = uint16(value)
		return nil
	case *uint32:
		if typed == nil {
			return errScanTarget
		}
		value, err := parseUnsignedToken(token, 32)
		if err != nil {
			return err
		}
		*typed = uint32(value)
		return nil
	case *uint64:
		if typed == nil {
			return errScanTarget
		}
		value, err := parseUnsignedToken(token, 64)
		if err != nil {
			return err
		}
		*typed = value
		return nil
	case *uintptr:
		if typed == nil {
			return errScanTarget
		}
		value, err := parseUnsignedToken(token, uintBitSize())
		if err != nil {
			return err
		}
		*typed = uintptr(value)
		return nil
	}

	return errScanTarget
}

func parseBoolToken(token string) (bool, error) {
	if equalFoldASCII(token, "true") || equalFoldASCII(token, "t") || token == "1" {
		return true, nil
	}
	if equalFoldASCII(token, "false") || equalFoldASCII(token, "f") || token == "0" {
		return false, nil
	}

	return false, errScanSyntax
}

func parseSignedToken(token string, bits uint) (int64, error) {
	if token == "" {
		return 0, errScanSyntax
	}

	negative := false
	switch token[0] {
	case '+':
		token = token[1:]
	case '-':
		negative = true
		token = token[1:]
	}
	if token == "" {
		return 0, errScanSyntax
	}

	base := uint64(10)
	if len(token) > 2 && token[0] == '0' && (token[1] == 'x' || token[1] == 'X') {
		base = 16
		token = token[2:]
	}
	if token == "" {
		return 0, errScanSyntax
	}

	limit := maxSignedMagnitude(bits, negative)

	value, err := parseUnsignedWithBase(token, base, limit)
	if err != nil {
		return 0, err
	}
	if negative {
		if value == uint64(1)<<(bits-1) {
			return -int64(value), nil
		}
		return -int64(value), nil
	}

	return int64(value), nil
}

func parseUnsignedToken(token string, bits uint) (uint64, error) {
	if token == "" {
		return 0, errScanSyntax
	}
	if token[0] == '+' {
		token = token[1:]
	}
	if token == "" || token[0] == '-' {
		return 0, errScanSyntax
	}

	base := uint64(10)
	if len(token) > 2 && token[0] == '0' && (token[1] == 'x' || token[1] == 'X') {
		base = 16
		token = token[2:]
	}
	if token == "" {
		return 0, errScanSyntax
	}

	return parseUnsignedWithBase(token, base, maxUnsignedForBits(bits))
}

func maxSignedMagnitude(bits uint, negative bool) uint64 {
	if bits >= 64 {
		if negative {
			return uint64(1) << 63
		}

		return (uint64(1) << 63) - 1
	}
	if negative {
		return uint64(1) << (bits - 1)
	}

	return (uint64(1) << (bits - 1)) - 1
}

func maxUnsignedForBits(bits uint) uint64 {
	if bits >= 64 {
		return ^uint64(0)
	}

	return (uint64(1) << bits) - 1
}

func parseUnsignedWithBase(token string, base uint64, limit uint64) (uint64, error) {
	value := uint64(0)

	for index := 0; index < len(token); index++ {
		digit, ok := digitValue(token[index])
		if !ok || digit >= base {
			return 0, errScanSyntax
		}

		next := value*base + digit
		if next < value || next > limit {
			return 0, errScanSyntax
		}
		value = next
	}

	return value, nil
}

func digitValue(value byte) (uint64, bool) {
	switch {
	case value >= '0' && value <= '9':
		return uint64(value - '0'), true
	case value >= 'a' && value <= 'f':
		return uint64(value-'a') + 10, true
	case value >= 'A' && value <= 'F':
		return uint64(value-'A') + 10, true
	}

	return 0, false
}

func equalFoldASCII(left string, right string) bool {
	if len(left) != len(right) {
		return false
	}

	for index := 0; index < len(left); index++ {
		if lowerASCII(left[index]) != lowerASCII(right[index]) {
			return false
		}
	}

	return true
}

func lowerASCII(value byte) byte {
	if value >= 'A' && value <= 'Z' {
		return value + ('a' - 'A')
	}

	return value
}

func intBitSize() uint {
	return uintBitSize()
}

func uintBitSize() uint {
	return uint(^uint(0)>>63)*32 + 32
}

var decimalDigits = [...]string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"}
var lowerHexDigits = [...]string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "a", "b", "c", "d", "e", "f"}
var upperHexDigits = [...]string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "A", "B", "C", "D", "E", "F"}
var decimalPowers = [...]uint64{
	10000000000000000000,
	1000000000000000000,
	100000000000000000,
	10000000000000000,
	1000000000000000,
	100000000000000,
	10000000000000,
	1000000000000,
	100000000000,
	10000000000,
	1000000000,
	100000000,
	10000000,
	1000000,
	100000,
	10000,
	1000,
	100,
	10,
	1,
}

func formatValue(value interface{}, verb byte) string {
	return formatValueSpec(value, formatSpec{verb: verb})
}

func formatValueSpec(value interface{}, spec formatSpec) string {
	if value == nil {
		if spec.verb == 'v' || spec.verb == 's' || spec.verb == 'T' {
			return applyFormatWidth("<nil>", spec, false)
		}

		return invalidNilVerb(spec.verb)
	}
	if spec.verb == 'T' {
		return applyFormatWidth(typeName(value), spec, false)
	}

	switch typed := value.(type) {
	case string:
		return formatStringValue(typed, spec)
	case []byte:
		return formatBytesValue(typed, spec)
	case []string:
		return formatStringSliceValue(typed, spec)
	case []bool:
		return formatBoolSliceValue(typed, spec)
	case []int:
		return formatIntSliceValue(typed, spec)
	case []int8:
		return formatInt8SliceValue(typed, spec)
	case []int16:
		return formatInt16SliceValue(typed, spec)
	case []int32:
		return formatInt32SliceValue(typed, spec)
	case []int64:
		return formatInt64SliceValue(typed, spec)
	case []uint:
		return formatUintSliceValue(typed, spec)
	case []uint16:
		return formatUint16SliceValue(typed, spec)
	case []uint32:
		return formatUint32SliceValue(typed, spec)
	case []uint64:
		return formatUint64SliceValue(typed, spec)
	case []uintptr:
		return formatUintptrSliceValue(typed, spec)
	case []error:
		return formatErrorSliceValue(typed, spec)
	case []interface{}:
		return formatInterfaceSliceValue(typed, spec)
	case bool:
		if spec.verb == 't' || spec.verb == 'v' {
			return formatBool(typed)
		}
	case int:
		return formatSignedValue(int64(typed), spec)
	case int8:
		return formatSignedValue(int64(typed), spec)
	case int16:
		return formatSignedValue(int64(typed), spec)
	case int32:
		return formatSignedValue(int64(typed), spec)
	case int64:
		return formatSignedValue(typed, spec)
	case uint:
		return formatUnsignedValue(uint64(typed), spec)
	case uint8:
		return formatUnsignedValue(uint64(typed), spec)
	case uint16:
		return formatUnsignedValue(uint64(typed), spec)
	case uint32:
		return formatUnsignedValue(uint64(typed), spec)
	case uint64:
		return formatUnsignedValue(typed, spec)
	case uintptr:
		return formatUnsignedValue(uint64(typed), spec)
	}

	if err, ok := value.(error); ok {
		switch spec.verb {
		case 'v', 's', 'q', 'x', 'X':
			return formatStringValue(err.Error(), spec)
		}
	}

	if stringer, ok := value.(Stringer); ok {
		switch spec.verb {
		case 'v', 's', 'q', 'x', 'X':
			return formatStringValue(stringer.String(), spec)
		}
	}

	return unsupportedVerb(spec.verb)
}

func formatStringValue(value string, spec formatSpec) string {
	switch spec.verb {
	case 's', 'v':
		return applyFormatWidth(value, spec, false)
	case 'q':
		if spec.sharp && canBackquote(value) {
			return applyFormatWidth("`"+value+"`", spec, false)
		}
		return applyFormatWidth(formatQuotedString(value, spec.plus), spec, false)
	case 'x':
		return applyFormatWidth(formatHexBytes([]byte(value), false), spec, false)
	case 'X':
		return applyFormatWidth(formatHexBytes([]byte(value), true), spec, false)
	}

	return unsupportedVerb(spec.verb)
}

func formatBytesValue(value []byte, spec formatSpec) string {
	switch spec.verb {
	case 's':
		return applyFormatWidth(string(value), spec, false)
	case 'v':
		return applyFormatWidth(formatUint8List(value), spec, false)
	case 'q':
		if spec.sharp && canBackquote(string(value)) {
			return applyFormatWidth("`"+string(value)+"`", spec, false)
		}
		return applyFormatWidth(formatQuotedString(string(value), spec.plus), spec, false)
	case 'x':
		return applyFormatWidth(formatHexBytes(value, false), spec, false)
	case 'X':
		return applyFormatWidth(formatHexBytes(value, true), spec, false)
	}

	return unsupportedVerb(spec.verb)
}

func formatStringSliceValue(value []string, spec formatSpec) string {
	if spec.verb != 'v' {
		return unsupportedVerb(spec.verb)
	}

	return applyFormatWidth(formatList(len(value), func(index int) string {
		return value[index]
	}), spec, false)
}

func formatBoolSliceValue(value []bool, spec formatSpec) string {
	if spec.verb != 'v' {
		return unsupportedVerb(spec.verb)
	}

	return applyFormatWidth(formatList(len(value), func(index int) string {
		return formatBool(value[index])
	}), spec, false)
}

func formatIntSliceValue(value []int, spec formatSpec) string {
	if spec.verb != 'v' {
		return unsupportedVerb(spec.verb)
	}

	return applyFormatWidth(formatList(len(value), func(index int) string {
		return formatInt64Decimal(int64(value[index]))
	}), spec, false)
}

func formatInt8SliceValue(value []int8, spec formatSpec) string {
	if spec.verb != 'v' {
		return unsupportedVerb(spec.verb)
	}

	return applyFormatWidth(formatList(len(value), func(index int) string {
		return formatInt64Decimal(int64(value[index]))
	}), spec, false)
}

func formatInt16SliceValue(value []int16, spec formatSpec) string {
	if spec.verb != 'v' {
		return unsupportedVerb(spec.verb)
	}

	return applyFormatWidth(formatList(len(value), func(index int) string {
		return formatInt64Decimal(int64(value[index]))
	}), spec, false)
}

func formatInt32SliceValue(value []int32, spec formatSpec) string {
	if spec.verb != 'v' {
		return unsupportedVerb(spec.verb)
	}

	return applyFormatWidth(formatList(len(value), func(index int) string {
		return formatInt64Decimal(int64(value[index]))
	}), spec, false)
}

func formatInt64SliceValue(value []int64, spec formatSpec) string {
	if spec.verb != 'v' {
		return unsupportedVerb(spec.verb)
	}

	return applyFormatWidth(formatList(len(value), func(index int) string {
		return formatInt64Decimal(value[index])
	}), spec, false)
}

func formatUintSliceValue(value []uint, spec formatSpec) string {
	if spec.verb != 'v' {
		return unsupportedVerb(spec.verb)
	}

	return applyFormatWidth(formatList(len(value), func(index int) string {
		return formatUint64Decimal(uint64(value[index]))
	}), spec, false)
}

func formatUint16SliceValue(value []uint16, spec formatSpec) string {
	if spec.verb != 'v' {
		return unsupportedVerb(spec.verb)
	}

	return applyFormatWidth(formatList(len(value), func(index int) string {
		return formatUint64Decimal(uint64(value[index]))
	}), spec, false)
}

func formatUint32SliceValue(value []uint32, spec formatSpec) string {
	if spec.verb != 'v' {
		return unsupportedVerb(spec.verb)
	}

	return applyFormatWidth(formatList(len(value), func(index int) string {
		return formatUint64Decimal(uint64(value[index]))
	}), spec, false)
}

func formatUint64SliceValue(value []uint64, spec formatSpec) string {
	if spec.verb != 'v' {
		return unsupportedVerb(spec.verb)
	}

	return applyFormatWidth(formatList(len(value), func(index int) string {
		return formatUint64Decimal(value[index])
	}), spec, false)
}

func formatUintptrSliceValue(value []uintptr, spec formatSpec) string {
	if spec.verb != 'v' {
		return unsupportedVerb(spec.verb)
	}

	return applyFormatWidth(formatList(len(value), func(index int) string {
		return formatUint64Decimal(uint64(value[index]))
	}), spec, false)
}

func formatErrorSliceValue(value []error, spec formatSpec) string {
	if spec.verb != 'v' {
		return unsupportedVerb(spec.verb)
	}

	return applyFormatWidth(formatList(len(value), func(index int) string {
		if value[index] == nil {
			return "<nil>"
		}

		return value[index].Error()
	}), spec, false)
}

func formatInterfaceSliceValue(value []interface{}, spec formatSpec) string {
	if spec.verb != 'v' {
		return unsupportedVerb(spec.verb)
	}

	return applyFormatWidth(formatList(len(value), func(index int) string {
		return formatValue(value[index], 'v')
	}), spec, false)
}

func formatUint8List(value []uint8) string {
	return formatList(len(value), func(index int) string {
		return formatUint64Decimal(uint64(value[index]))
	})
}

func formatList(length int, render func(index int) string) string {
	buffer := &buffer{}
	buffer.writeByte('[')
	for index := 0; index < length; index++ {
		if index > 0 {
			buffer.writeByte(' ')
		}
		buffer.writeString(render(index))
	}
	buffer.writeByte(']')
	return buffer.String()
}

func formatSignedValue(value int64, spec formatSpec) string {
	switch spec.verb {
	case 'd', 'v':
		return applyFormatWidth(formatInt64Decimal(value), spec, true)
	case 'q':
		return applyFormatWidth(formatQuotedSignedRune(value, spec.plus), spec, false)
	case 'x':
		return applyFormatWidth(formatUint64Hex(uint64(value), lowerHexDigits[:]), spec, true)
	case 'X':
		return applyFormatWidth(formatUint64Hex(uint64(value), upperHexDigits[:]), spec, true)
	case 'c':
		return applyFormatWidth(string([]byte{byte(value)}), spec, false)
	}

	return unsupportedVerb(spec.verb)
}

func formatUnsignedValue(value uint64, spec formatSpec) string {
	switch spec.verb {
	case 'd', 'v':
		return applyFormatWidth(formatUint64Decimal(value), spec, true)
	case 'q':
		return applyFormatWidth(formatQuotedUnsignedRune(value, spec.plus), spec, false)
	case 'x':
		return applyFormatWidth(formatUint64Hex(value, lowerHexDigits[:]), spec, true)
	case 'X':
		return applyFormatWidth(formatUint64Hex(value, upperHexDigits[:]), spec, true)
	case 'c':
		return applyFormatWidth(string([]byte{byte(value)}), spec, false)
	}

	return unsupportedVerb(spec.verb)
}

func applyFormatWidth(value string, spec formatSpec, numeric bool) string {
	if spec.width <= len(value) {
		return value
	}

	padByte := byte(' ')
	if spec.zero && numeric {
		padByte = '0'
	}
	padCount := spec.width - len(value)
	if padByte != '0' {
		return repeatFormatByte(padByte, padCount) + value
	}
	if len(value) > 0 && value[0] == '-' {
		return "-" + repeatFormatByte('0', padCount) + value[1:]
	}

	return repeatFormatByte('0', padCount) + value
}

func repeatFormatByte(value byte, count int) string {
	if count <= 0 {
		return ""
	}

	data := make([]byte, count)
	for index := 0; index < count; index++ {
		data[index] = value
	}

	return string(data)
}

func isFormatDigit(value byte) bool {
	return value >= '0' && value <= '9'
}

func formatBool(value bool) string {
	if value {
		return "true"
	}

	return "false"
}

func formatInt64Decimal(value int64) string {
	if value < 0 {
		return "-" + formatUint64Decimal(uint64(^value)+1)
	}

	return formatUint64Decimal(uint64(value))
}

func formatUint64Decimal(value uint64) string {
	if value == 0 {
		return "0"
	}

	text := ""
	started := false

	for index := 0; index < len(decimalPowers); index++ {
		digit := uint32(0)
		for value >= decimalPowers[index] {
			value -= decimalPowers[index]
			digit++
		}

		if digit != 0 || started {
			text += decimalDigits[digit]
			started = true
		}
	}

	return text
}

func formatUint64Hex(value uint64, digits []string) string {
	if value == 0 {
		return "0"
	}

	text := ""
	started := false

	for shift := uint(60); ; shift -= 4 {
		digit := uint32((value >> shift) & 0x0F)
		if digit != 0 || started {
			text += digits[digit]
			started = true
		}

		if shift == 0 {
			break
		}
	}

	return text
}

func formatHexBytes(value []byte, upper bool) string {
	if len(value) == 0 {
		return ""
	}

	digits := lowerHexDigits[:]
	if upper {
		digits = upperHexDigits[:]
	}

	text := ""
	for index := 0; index < len(value); index++ {
		text += digits[uint32(value[index]>>4)]
		text += digits[uint32(value[index]&0x0F)]
	}

	return text
}

func formatQuotedString(value string, asciiOnly bool) string {
	buffer := &buffer{}
	buffer.writeByte('"')

	for len(value) > 0 {
		r, size := decodeQuotedRuneInString(value)
		if size == 0 {
			break
		}
		if size == 1 && r == quotedRuneError && value[0] >= quotedRuneSelf {
			appendHexEscape(buffer, value[0])
			value = value[1:]
			continue
		}

		appendEscapedRune(buffer, r, '"', asciiOnly)
		value = value[size:]
	}

	buffer.writeByte('"')
	return buffer.String()
}

func formatQuotedSignedRune(value int64, asciiOnly bool) string {
	if value < 0 || value > int64(quotedMaxRune) {
		return formatQuotedRuneLiteral(quotedRuneError, asciiOnly)
	}

	return formatQuotedRuneLiteral(rune(value), asciiOnly)
}

func formatQuotedUnsignedRune(value uint64, asciiOnly bool) string {
	if value > uint64(quotedMaxRune) {
		return formatQuotedRuneLiteral(quotedRuneError, asciiOnly)
	}

	return formatQuotedRuneLiteral(rune(value), asciiOnly)
}

func formatQuotedRuneLiteral(value rune, asciiOnly bool) string {
	if !validQuotedRune(value) {
		value = quotedRuneError
	}

	buffer := &buffer{}
	buffer.writeByte('\'')
	appendEscapedRune(buffer, value, '\'', asciiOnly)
	buffer.writeByte('\'')
	return buffer.String()
}

func appendEscapedRune(buffer *buffer, value rune, quote byte, asciiOnly bool) {
	if value == rune(quote) || value == '\\' {
		buffer.writeByte('\\')
		buffer.writeByte(byte(value))
		return
	}
	if isPrintableQuotedRune(value, asciiOnly) {
		buffer.data = appendQuotedRuneUTF8(buffer.data, value)
		return
	}

	switch value {
	case '\a':
		buffer.writeString(`\a`)
	case '\b':
		buffer.writeString(`\b`)
	case '\f':
		buffer.writeString(`\f`)
	case '\n':
		buffer.writeString(`\n`)
	case '\r':
		buffer.writeString(`\r`)
	case '\t':
		buffer.writeString(`\t`)
	case '\v':
		buffer.writeString(`\v`)
	default:
		switch {
		case value < ' ' || value == 0x7F:
			appendHexEscape(buffer, byte(value))
		case value < 0x10000:
			appendUnicodeEscape(buffer, 'u', uint32(value), 4)
		default:
			appendUnicodeEscape(buffer, 'U', uint32(value), 8)
		}
	}
}

func isPrintableQuotedRune(value rune, asciiOnly bool) bool {
	switch {
	case !validQuotedRune(value):
		return false
	case asciiOnly:
		return value < quotedRuneSelf && quotedIsPrint(value)
	default:
		return quotedIsPrint(value)
	}
}

func canBackquote(value string) bool {
	for len(value) > 0 {
		r, size := decodeQuotedRuneInString(value)
		value = value[size:]
		if size > 1 {
			if r == '\ufeff' {
				return false
			}
			continue
		}
		if r == quotedRuneError {
			return false
		}
		if (r < ' ' && r != '\t') || r == '`' || r == '\u007f' {
			return false
		}
	}

	return true
}

func appendHexEscape(buffer *buffer, value byte) {
	buffer.writeByte('\\')
	buffer.writeByte('x')
	buffer.writeString(lowerHexDigits[value>>4])
	buffer.writeString(lowerHexDigits[value&0x0F])
}

func appendUnicodeEscape(buffer *buffer, prefix byte, value uint32, digits int) {
	buffer.writeByte('\\')
	buffer.writeByte(prefix)

	for shift := uint((digits - 1) * 4); ; shift -= 4 {
		buffer.writeString(lowerHexDigits[(value>>shift)&0x0F])
		if shift == 0 {
			break
		}
	}
}

const (
	quotedRuneError = '\uFFFD'
	quotedRuneSelf  = 0x80
	quotedMaxRune   = '\U0010FFFF'
	quotedUTFMax    = 4

	quotedSurrogateMin = 0xD800
	quotedSurrogateMax = 0xDFFF
)

func decodeQuotedRuneInString(value string) (rune, int) {
	if len(value) == 0 {
		return quotedRuneError, 0
	}
	if value[0] < quotedRuneSelf {
		return rune(value[0]), 1
	}

	need := quotedSequenceLength(value[0])
	if need < 0 || len(value) < need {
		return quotedRuneError, 1
	}
	if !validQuotedSequenceString(value, need) {
		return quotedRuneError, 1
	}

	switch need {
	case 2:
		return rune(value[0]&0x1F)<<6 | rune(value[1]&0x3F), 2
	case 3:
		return rune(value[0]&0x0F)<<12 |
			rune(value[1]&0x3F)<<6 |
			rune(value[2]&0x3F), 3
	case 4:
		return rune(value[0]&0x07)<<18 |
			rune(value[1]&0x3F)<<12 |
			rune(value[2]&0x3F)<<6 |
			rune(value[3]&0x3F), 4
	default:
		return quotedRuneError, 1
	}
}

func appendQuotedRuneUTF8(dst []byte, value rune) []byte {
	var encoded [quotedUTFMax]byte

	count := encodeQuotedRune(encoded[:], value)
	return append(dst, encoded[:count]...)
}

func encodeQuotedRune(dst []byte, value rune) int {
	if !validQuotedRune(value) {
		value = quotedRuneError
	}

	switch {
	case value <= 0x7F:
		dst[0] = byte(value)
		return 1
	case value <= 0x7FF:
		dst[0] = 0xC0 | byte(value>>6)
		dst[1] = 0x80 | byte(value)&0x3F
		return 2
	case value <= 0xFFFF:
		dst[0] = 0xE0 | byte(value>>12)
		dst[1] = 0x80 | byte(value>>6)&0x3F
		dst[2] = 0x80 | byte(value)&0x3F
		return 3
	default:
		dst[0] = 0xF0 | byte(value>>18)
		dst[1] = 0x80 | byte(value>>12)&0x3F
		dst[2] = 0x80 | byte(value>>6)&0x3F
		dst[3] = 0x80 | byte(value)&0x3F
		return 4
	}
}

func validQuotedRune(value rune) bool {
	return value >= 0 && value <= quotedMaxRune && (value < quotedSurrogateMin || value > quotedSurrogateMax)
}

func quotedSequenceLength(value byte) int {
	switch {
	case value < quotedRuneSelf:
		return 1
	case value < 0xC2:
		return -1
	case value < 0xE0:
		return 2
	case value < 0xF0:
		return 3
	case value <= 0xF4:
		return 4
	default:
		return -1
	}
}

func validQuotedSequenceString(value string, need int) bool {
	switch need {
	case 2:
		return isQuotedContinuation(value[1])
	case 3:
		return validQuotedSecondByte3(value[0], value[1]) &&
			isQuotedContinuation(value[2])
	case 4:
		return validQuotedSecondByte4(value[0], value[1]) &&
			isQuotedContinuation(value[2]) &&
			isQuotedContinuation(value[3])
	default:
		return false
	}
}

func validQuotedSecondByte3(first byte, second byte) bool {
	switch first {
	case 0xE0:
		return second >= 0xA0 && second <= 0xBF
	case 0xED:
		return second >= 0x80 && second <= 0x9F
	default:
		return isQuotedContinuation(second)
	}
}

func validQuotedSecondByte4(first byte, second byte) bool {
	switch first {
	case 0xF0:
		return second >= 0x90 && second <= 0xBF
	case 0xF4:
		return second >= 0x80 && second <= 0x8F
	default:
		return isQuotedContinuation(second)
	}
}

func isQuotedContinuation(value byte) bool {
	return value&0xC0 == 0x80
}

func missingVerb(verb byte) string {
	return "%!" + string([]byte{verb}) + "(MISSING)"
}

func invalidNilVerb(verb byte) string {
	return "%!" + string([]byte{verb}) + "(<nil>)"
}

func unsupportedVerb(verb byte) string {
	return "%!" + string([]byte{verb}) + "(UNSUPPORTED)"
}

func renderWrapValue(value interface{}, allowWrap bool, wrapped *[]error) string {
	err, ok := value.(error)
	if !ok || !allowWrap {
		return invalidWrapValue(value)
	}

	*wrapped = append(*wrapped, err)
	return formatValue(value, 'v')
}

func invalidWrapValue(value interface{}) string {
	if value == nil {
		return "%!w(<nil>)"
	}

	return "%!w(" + typeName(value) + "=" + formatValue(value, 'v') + ")"
}

func wrapTypeName(value interface{}) string {
	return typeName(value)
}

func typeName(value interface{}) string {
	if value == nil {
		return "<nil>"
	}
	typ := reflectlite.TypeOf(value)
	if typ == nil {
		return "value"
	}
	name := typ.String()
	if name == "" {
		return "value"
	}
	return name
}
