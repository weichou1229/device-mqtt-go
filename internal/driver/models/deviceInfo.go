package models

import "github.com/edgexfoundry/go-mod-core-contracts/v2/models"

type DeviceInfo struct {
	Name        string                               `json:"name"`
	ProfileName string                               `json:"profile"`
	Protocols   map[string]models.ProtocolProperties `json:"protocols"`
	Properties  map[string]interface{}               `json:"properties"`
}
