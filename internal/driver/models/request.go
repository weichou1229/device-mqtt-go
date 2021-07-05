package models

import (
	"github.com/edgexfoundry/device-sdk-go/v2/pkg/service"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/common"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/models"
	"github.com/google/uuid"
	"strconv"
	"strings"
)

const (
	ProfileAddOperation = "profile:add"
	DeviceAddOperation  = "device:add"
)

type ProfileRequest struct {
	Client    string        `json:"client"`
	RequestId string        `json:"request_id"`
	Op        string        `json:"op"`
	Profile   DeviceProfile `json:"profile"`
}

type DeviceRequest struct {
	Client     string     `json:"client"`
	RequestId  string     `json:"request_id"`
	Op         string     `json:"op"`
	DeviceName string     `json:"device"`
	DeviceInfo DeviceInfo `json:"device_info"`
}

func NewProfileRequest(profile models.DeviceProfile) ProfileRequest {
	ds := service.RunningService()
	profileRequest := ProfileRequest{
		Client:    ds.ServiceName,
		RequestId: uuid.New().String(),
		Op:        ProfileAddOperation,
		Profile: DeviceProfile{
			DescribedObject: DescribedObject{},
			Name:            profile.Name,
			Manufacturer:    profile.Manufacturer,
			Model:           profile.Model,
			Labels:          profile.Labels,
			DeviceResources: nil,
			DeviceCommands:  nil,
		},
	}

	profileRequest.Profile.DeviceResources = DeviceResources(profile.DeviceResources)
	profileRequest.Profile.DeviceCommands = DeviceCommands(profile.DeviceCommands)

	return profileRequest
}

func NewDeviceRequest(device models.Device) DeviceRequest {
	ds := service.RunningService()
	deviceRepuest := DeviceRequest{
		Client:     ds.ServiceName,
		RequestId:  uuid.New().String(),
		Op:         DeviceAddOperation,
		DeviceName: device.Name,
		DeviceInfo: DeviceInfo{
			ProfileName: device.ProfileName,
			Protocols:   device.Protocols,
		},
	}
	return deviceRepuest
}

func DeviceResources(deviceResources []models.DeviceResource) []DeviceResource {
	resources := make([]DeviceResource, len(deviceResources))
	for i, r := range deviceResources {
		resources[i] = DeviceResource{
			Name:        r.Name,
			Description: r.Description,
			Tag:         r.Tag,
			Attributes:  r.Attributes,
			Properties: ProfileProperty{
				Value: PropertyValue{
					Type:          r.Properties.ValueType,
					ReadWrite:     r.Properties.ReadWrite,
					Minimum:       r.Properties.Minimum,
					Maximum:       r.Properties.Maximum,
					DefaultValue:  r.Properties.DefaultValue,
					Mask:          r.Properties.Mask,
					Shift:         r.Properties.Shift,
					Scale:         r.Properties.Scale,
					Offset:        r.Properties.Offset,
					Base:          r.Properties.Base,
					Assertion:     r.Properties.Assertion,
					FloatEncoding: "eNotation",
					MediaType:     r.Properties.MediaType,
				},
				Units: Units{
					Type:         "String",
					ReadWrite:    common.ReadWrite_RW,
					DefaultValue: r.Properties.Units,
				},
			},
		}
	}

	return resources
}

func DeviceCommands(deviceCommands []models.DeviceCommand) []ProfileResource {
	commands := make([]ProfileResource, len(deviceCommands))
	for i, c := range deviceCommands {
		commands[i] = ProfileResource{
			Name: c.Name,
		}
		if strings.Contains(c.ReadWrite, common.ReadWrite_R) {
			commands[i].Get = Operation(ResourceOperationGet, c.ResourceOperations)
		}
		if strings.Contains(c.ReadWrite, common.ReadWrite_W) {
			commands[i].Set = Operation(ResourceOperationSet, c.ResourceOperations)
		}
	}

	return commands
}

func Operation(op string, resourceOperations []models.ResourceOperation) []ResourceOperation {
	operations := make([]ResourceOperation, len(resourceOperations))
	for i, ro := range resourceOperations {
		operations[i] = ResourceOperation{
			Index:          strconv.Itoa(i),
			Operation:      op,
			DeviceResource: ro.DeviceResource,
			Mappings:       ro.Mappings,
		}
	}
	return operations
}
