package driver

import (
	"encoding/json"
	"fmt"

	xrtModel "github.com/edgexfoundry/device-mqtt-go/internal/driver/models"

	"github.com/edgexfoundry/device-sdk-go/v2/pkg/service"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/errors"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/models"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func AddDeviceToXRT(mqttClient mqtt.Client, deviceName string, protocols map[string]models.ProtocolProperties) errors.EdgeX {
	ds := service.RunningService()
	device, err := ds.GetDeviceByName(deviceName)
	if err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}
	profile, err := ds.GetProfileByName(device.ProfileName)
	if err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}

	err = addProfileToXRT(mqttClient, profile)
	if err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}

	err = addDeviceToXRT(mqttClient, device)
	if err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}
	return nil
}

func addProfileToXRT(mqttClient mqtt.Client, profile models.DeviceProfile) errors.EdgeX {
	protocol := "Modbus"
	serviceName := "modbus_ds"
	topic := fmt.Sprintf("xrt/profile/%s/%s/request", protocol, serviceName)
	profileRequest := xrtModel.NewProfileAddRequest(profile)
	jsonData, err := json.Marshal(profileRequest)
	if err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}
	token := mqttClient.Publish(topic, byte(0), false, jsonData)
	if token.Wait() && token.Error() != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}

	cmdResponse, ok := driver.fetchCommandResponse(profileRequest.RequestId)
	if !ok {
		return errors.NewCommonEdgeX(errors.KindServerError, fmt.Sprintf("can not fetch command response for adding the profile `%s`", profile.Name), nil)
	}

	fmt.Printf("Profile %s added to XRT. Response: `%s` \n", profile.Name, cmdResponse)
	return nil
}

func addDeviceToXRT(mqttClient mqtt.Client, device models.Device) errors.EdgeX {
	protocol := "Modbus"
	serviceName := "modbus_ds"
	topic := fmt.Sprintf("xrt/device/%s/%s/request", protocol, serviceName)
	deviceRequest := xrtModel.NewDeviceAddRequest(device)
	jsonData, err := json.Marshal(deviceRequest)
	if err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}

	token := mqttClient.Publish(topic, byte(0), false, jsonData)
	if token.Wait() && token.Error() != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}

	cmdResponse, ok := driver.fetchCommandResponse(deviceRequest.RequestId)
	if !ok {
		return errors.NewCommonEdgeX(errors.KindServerError, fmt.Sprintf("can not fetch command response for adding the device `%s`", device.Name), nil)
	}
	fmt.Printf("Device %s added to XRT. Response: `%s` \n", device.Name, cmdResponse)
	return nil
}
