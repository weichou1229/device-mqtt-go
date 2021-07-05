package models

import (
	"encoding/json"
)

// Action describes state related to the capabilities of a device
type Action struct {
	Path      string     `json:"path,omitempty" yaml:"path,omitempty"`           // Path used by service for action on a device or sensor
	Responses []Response `json:"responses,omitempty" yaml:"responses,omitempty"` // Responses from get or put requests to service
	URL       string     `json:"url,omitempty" yaml:"url,omitempty"`             // Url for requests from command service
}

// String returns a JSON formatted string representation of the Action
func (a Action) String() string {
	out, err := json.Marshal(a)
	if err != nil {
		return err.Error()
	}
	return string(out)
}
