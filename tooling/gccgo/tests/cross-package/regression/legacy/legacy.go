package legacy

// This ordinary package is compiled without the generic compiler stage.
type state byte

type Record struct {
	Value  int
	hidden state
}
