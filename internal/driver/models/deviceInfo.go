package models

import "github.com/edgexfoundry/go-mod-core-contracts/v2/models"

type DeviceInfo struct {
	ProfileName string                               `json:"profile"`
	Protocols   map[string]models.ProtocolProperties `json:"protocols"`
}
