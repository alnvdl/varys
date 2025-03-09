package list

// InputFeed represents a feed that can be added to the list (e.g., via an
// external import process).
type InputFeed struct {
	Name   string `json:"name"`
	URL    string `json:"url"`
	Type   string `json:"type"`
	Params any    `json:"params"`
}
