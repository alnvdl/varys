package fetch

import (
	"encoding/json"
	"fmt"
)

// feedParams is to be implemented by specific feed params structs.
type feedParams interface {
	validate() error
}

// parseParams takes feed params, unmarshalls them into the dst struct and
// validates them.
func parseParams(params any, dst feedParams) error {
	data, err := json.Marshal(params)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, dst)
	if err != nil {
		return fmt.Errorf("cannot unmarshal: %v", err)
	}
	if err := dst.validate(); err != nil {
		return fmt.Errorf("cannot validate: %v", err)
	}
	return nil
}
