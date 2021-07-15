package models

import (
	"strconv"
	"strings"

	sdkModel "github.com/edgexfoundry/device-sdk-go/v2/pkg/models"
	"github.com/edgexfoundry/device-sdk-go/v2/pkg/service"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/common"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/models"

	"github.com/google/uuid"
)

const (
	ProfileAddOperation  = "profile:add"
	ProfileListOperation = "profile:list"
	ProfileGetOperation  = "profile:read"

	DeviceAddOperation         = "device:add"
	DeviceResourceGetOperation = "device:get"
	DeviceGetOperation         = "device:read"
	DeviceSetOperation         = "device:put"
	DeviceListOperation        = "device:list"

	DiscoveryTriggerOperation = "discovery:trigger"
)

type XRTRequest struct {
	Client    string `json:"client"`
	RequestId string `json:"request_id"`
	Op        string `json:"op"`
}

type ProfileAddRequest struct {
	XRTRequest `json:",inline"`
	Profile    DeviceProfile `json:"profile"`
}

type ProfileGetRequest struct {
	XRTRequest `json:",inline"`
	Profile    string `json:"profile"`
}

type DeviceAddRequest struct {
	XRTRequest `json:",inline"`
	DeviceName string     `json:"device"`
	DeviceInfo DeviceInfo `json:"device_info"`
}

type DeviceGetRequest struct {
	XRTRequest `json:",inline"`
	Device     string `json:"device"`
}

type DeviceResourceGetRequest struct {
	XRTRequest `json:",inline"`
	DeviceName string   `json:"device"`
	Resource   []string `json:"resource"`
}

type DeviceResourceSetRequest struct {
	XRTRequest `json:",inline"`
	DeviceName string                 `json:"device"`
	Values     map[string]interface{} `json:"values"`
}

func NewXRTRequest(op string) XRTRequest {
	ds := service.RunningService()
	return XRTRequest{
		Client:    ds.ServiceName,
		RequestId: uuid.New().String(),
		Op:        op,
	}
}

func NewProfileAddRequest(profile models.DeviceProfile) ProfileAddRequest {
	ds := service.RunningService()
	req := ProfileAddRequest{
		XRTRequest: XRTRequest{
			Client:    ds.ServiceName,
			RequestId: uuid.New().String(),
			Op:        ProfileAddOperation,
		},
		Profile: profileDTO(profile),
	}

	return req
}

func NewProfileGetRequest(profileName string) ProfileGetRequest {
	ds := service.RunningService()
	return ProfileGetRequest{
		XRTRequest: XRTRequest{
			Client:    ds.ServiceName,
			RequestId: uuid.New().String(),
			Op:        ProfileGetOperation,
		},
		Profile: profileName,
	}
}

func NewDeviceAddRequest(device models.Device) DeviceAddRequest {
	ds := service.RunningService()
	deviceRequest := DeviceAddRequest{
		XRTRequest: XRTRequest{
			Client:    ds.ServiceName,
			RequestId: uuid.New().String(),
			Op:        DeviceAddOperation,
		},
		DeviceName: device.Name,
		DeviceInfo: DeviceInfo{
			ProfileName: device.ProfileName,
			Protocols:   device.Protocols,
		},
	}
	return deviceRequest
}

func NewDeviceGetRequest(deviceName string) DeviceGetRequest {
	ds := service.RunningService()
	req := DeviceGetRequest{
		XRTRequest: XRTRequest{
			Client:    ds.ServiceName,
			RequestId: uuid.New().String(),
			Op:        DeviceGetOperation,
		},
		Device: deviceName,
	}
	return req
}

func NewDeviceResourceGetRequest(deviceName string, reqs []sdkModel.CommandRequest) DeviceResourceGetRequest {
	ds := service.RunningService()
	req := DeviceResourceGetRequest{
		XRTRequest: XRTRequest{
			Client:    ds.ServiceName,
			RequestId: uuid.New().String(),
			Op:        DeviceResourceGetOperation,
		},
		DeviceName: deviceName,
		Resource:   nil,
	}
	resources := make([]string, len(reqs))
	for i, req := range reqs {
		//resources[i] = req.DeviceResourceName
		resources[i] = strings.Replace(req.DeviceResourceName, "_", ":", -1) // TODO Invalid Resource Name  SimpleServer:object-identifier
	}
	req.Resource = resources
	return req
}

func NewDeviceResourceSetRequest(deviceName string, reqs []sdkModel.CommandRequest, params []*sdkModel.CommandValue) DeviceResourceSetRequest {
	ds := service.RunningService()
	req := DeviceResourceSetRequest{
		XRTRequest: XRTRequest{
			Client:    ds.ServiceName,
			RequestId: uuid.New().String(),
			Op:        DeviceSetOperation,
		},
		DeviceName: deviceName,
		Values:     nil,
	}
	values := make(map[string]interface{}, len(reqs))
	for i, req := range reqs {
		values[req.DeviceResourceName] = params[i].Value
	}
	req.Values = values
	return req
}

func profileDTO(profile models.DeviceProfile) DeviceProfile {
	dto := DeviceProfile{
		DescribedObject: DescribedObject{},
		Name:            profile.Name,
		Manufacturer:    profile.Manufacturer,
		Model:           profile.Model,
		Labels:          profile.Labels,
		DeviceResources: nil,
		DeviceCommands:  nil,
	}
	dto.DeviceResources = DeviceResourcesDTO(profile.DeviceResources)
	dto.DeviceCommands = DeviceCommandsDTO(profile.DeviceCommands)
	return dto
}

func DeviceResourcesDTO(deviceResources []models.DeviceResource) []DeviceResource {
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

func DeviceCommandsDTO(deviceCommands []models.DeviceCommand) []ProfileResource {
	commands := make([]ProfileResource, len(deviceCommands))
	for i, c := range deviceCommands {
		commands[i] = ProfileResource{
			Name: c.Name,
		}
		if strings.Contains(c.ReadWrite, common.ReadWrite_R) {
			commands[i].Get = OperationDTO(ResourceOperationGet, c.ResourceOperations)
		}
		if strings.Contains(c.ReadWrite, common.ReadWrite_W) {
			commands[i].Set = OperationDTO(ResourceOperationSet, c.ResourceOperations)
		}
	}

	return commands
}

func OperationDTO(op string, resourceOperations []models.ResourceOperation) []ResourceOperation {
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

func Profile(profile DeviceProfile) models.DeviceProfile {
	dto := models.DeviceProfile{
		Description:     profile.Description,
		Name:            profile.Name,
		Manufacturer:    profile.Manufacturer,
		Model:           profile.Model,
		Labels:          profile.Labels,
		DeviceResources: nil,
		DeviceCommands:  nil,
	}
	dto.DeviceResources = DeviceResources(profile.DeviceResources)
	dto.DeviceCommands = DeviceCommands(profile.DeviceCommands)
	return dto
}

func DeviceResources(deviceResources []DeviceResource) []models.DeviceResource {
	resources := make([]models.DeviceResource, len(deviceResources))
	for i, r := range deviceResources {
		resources[i] = models.DeviceResource{
			Description: r.Description,
			Name:        strings.Replace(r.Name, ":", "_", -1), // TODO Invalid Resource Name  SimpleServer:object-identifier
			IsHidden:    false,
			Tag:         r.Tag,
			Properties: models.ResourceProperties{
				ValueType:    r.Properties.Value.Type,
				ReadWrite:    r.Properties.Value.ReadWrite,
				Units:        r.Properties.Units.DefaultValue,
				Minimum:      r.Properties.Value.Minimum,
				Maximum:      r.Properties.Value.Maximum,
				DefaultValue: r.Properties.Value.DefaultValue,
				Mask:         r.Properties.Value.Mask,
				Shift:        r.Properties.Value.Shift,
				Scale:        r.Properties.Value.Scale,
				Offset:       r.Properties.Value.Offset,
				Base:         r.Properties.Value.Base,
				Assertion:    r.Properties.Value.Assertion,
				MediaType:    r.Properties.Value.MediaType,
			},
			Attributes: r.Attributes,
		}
	}

	return resources
}

func DeviceCommands(deviceCommands []ProfileResource) []models.DeviceCommand {
	commands := make([]models.DeviceCommand, len(deviceCommands))
	for i, c := range deviceCommands {
		commands[i] = models.DeviceCommand{
			Name:               c.Name,
			IsHidden:           false,
			ReadWrite:          "",
			ResourceOperations: nil,
		}
		if len(c.Get) > 0 && len(c.Set) > 0 {
			commands[i].ReadWrite = common.ReadWrite_RW
		} else if len(c.Get) > 0 {
			commands[i].ReadWrite = common.ReadWrite_R
		} else {
			commands[i].ReadWrite = common.ReadWrite_W
		}
		var ros []models.ResourceOperation
		for _, op := range c.Get {
			ro := models.ResourceOperation{
				DeviceResource: op.DeviceResource,
				DefaultValue:   "",
				Mappings:       op.Mappings,
			}
			ros = append(ros, ro)
		}
		for _, op := range c.Set {
			exists := existFromResourceDTO(op.DeviceResource, ros)
			if exists {
				continue
			}
			ro := models.ResourceOperation{
				DeviceResource: op.DeviceResource,
				DefaultValue:   "",
				Mappings:       op.Mappings,
			}
			ros = append(ros, ro)
		}
		commands[i].ResourceOperations = ros
	}

	return commands
}

func existFromResourceDTO(resourceName string, ros []models.ResourceOperation) bool {
	for _, ro := range ros {
		if ro.DeviceResource == resourceName {
			return true
		}
	}
	return false
}

func Device(device DeviceInfo) models.Device {
	return models.Device{
		Name:           device.Name,
		Description:    "",
		AdminState:     models.Unlocked,
		OperatingState: models.Up,
		Protocols:      device.Protocols,
		LastConnected:  0,
		LastReported:   0,
		Labels:         nil,
		Location:       nil,
		ServiceName:    service.RunningService().ServiceName,
		ProfileName:    device.ProfileName,
		AutoEvents:     nil,
		Notify:         false,
	}
}
