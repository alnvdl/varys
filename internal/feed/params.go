package feed

import (
	"encoding/json"
	"fmt"
)

// feedParams is to be implemented by specific feed params structs.
type FeedParams interface {
	Validate() error
}

// parseParams takes feed params, unmarshalls them into the dst struct and
// validates them.
func ParseParams(params any, dst FeedParams) error {
	data, err := json.Marshal(params)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, dst)
	if err != nil {
		return fmt.Errorf("cannot unmarshal: %v", err)
	}
	if err := dst.Validate(); err != nil {
		return fmt.Errorf("cannot validate: %v", err)
	}
	return nil
}
