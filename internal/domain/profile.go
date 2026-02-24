package domain

// Profile is the runtime representation of a keyword profile used for domain scoring.
type Profile struct {
	Name           string
	Keywords       []string
	SuspiciousTLDs []string
	SkipSuffixes   []string
	Description    string
}
