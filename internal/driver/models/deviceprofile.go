package models

import (
	"encoding/json"
)

// DeviceProfile represents the attributes and operational capabilities of a device. It is a template for which
// there can be multiple matching devices within a given system.
type DeviceProfile struct {
	DescribedObject `yaml:",inline"`
	Id              string            `json:"id,omitempty" yaml:"id,omitempty"`
	Name            string            `json:"name,omitempty" yaml:"name,omitempty"`                 // Non-database identifier (must be unique)
	Manufacturer    string            `json:"manufacturer,omitempty" yaml:"manufacturer,omitempty"` // Manufacturer of the device
	Model           string            `json:"model,omitempty" yaml:"model,omitempty"`               // Model of the device
	Labels          []string          `json:"labels,omitempty" yaml:"labels,flow,omitempty"`        // Labels used to search for groups of profiles
	DeviceResources []DeviceResource  `json:"deviceResources,omitempty" yaml:"deviceResources,omitempty"`
	DeviceCommands  []ProfileResource `json:"deviceCommands,omitempty" yaml:"deviceCommands,omitempty"`
	CoreCommands    []Command         `json:"coreCommands,omitempty" yaml:"coreCommands,omitempty"` // List of commands to Get/Put information for devices associated with this profile
	isValidated     bool              // internal member used for validation check
}

// UnmarshalJSON implements the Unmarshaler interface for the DeviceProfile type
func (dp *DeviceProfile) UnmarshalJSON(data []byte) error {
	var err error
	type Alias struct {
		DescribedObject `json:",inline"`
		Id              *string           `json:"id"`
		Name            *string           `json:"name"`
		Manufacturer    *string           `json:"manufacturer"`
		Model           *string           `json:"model"`
		Labels          []string          `json:"labels"`
		DeviceResources []DeviceResource  `json:"deviceResources"`
		DeviceCommands  []ProfileResource `json:"deviceCommands"`
		CoreCommands    []Command         `json:"coreCommands"`
	}
	a := Alias{}
	// Error with unmarshaling
	if err = json.Unmarshal(data, &a); err != nil {
		return err
	}

	// Check nil fields
	if a.Id != nil {
		dp.Id = *a.Id
	}
	if a.Name != nil {
		dp.Name = *a.Name
	}
	if a.Manufacturer != nil {
		dp.Manufacturer = *a.Manufacturer
	}
	if a.Model != nil {
		dp.Model = *a.Model
	}
	dp.DescribedObject = a.DescribedObject
	dp.Labels = a.Labels
	dp.DeviceResources = a.DeviceResources
	dp.DeviceCommands = a.DeviceCommands
	dp.CoreCommands = a.CoreCommands
	return err

}

/*
 * To String function for DeviceProfile
 */
func (dp DeviceProfile) String() string {
	out, err := json.Marshal(dp)
	if err != nil {
		return err.Error()
	}
	return string(out)
}
