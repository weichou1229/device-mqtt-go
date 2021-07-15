// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2019-2021 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package driver

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	xrtModel "github.com/edgexfoundry/device-mqtt-go/internal/driver/models"

	sdkModel "github.com/edgexfoundry/device-sdk-go/v2/pkg/models"
	"github.com/edgexfoundry/device-sdk-go/v2/pkg/service"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/clients/logger"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/common"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/errors"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/models"

	"github.com/eclipse/paho.mqtt.golang"
	"github.com/spf13/cast"
)

var once sync.Once
var driver *Driver

type Driver struct {
	Logger           logger.LoggingClient
	AsyncCh          chan<- *sdkModel.AsyncValues
	CommandResponses sync.Map
	serviceConfig    *ServiceConfig
	mqttClient       mqtt.Client
}

const RequestTopic = "RequestTopic"

func NewProtocolDriver() sdkModel.ProtocolDriver {
	once.Do(func() {
		driver = new(Driver)
	})
	return driver
}

func (d *Driver) Initialize(lc logger.LoggingClient, asyncCh chan<- *sdkModel.AsyncValues, deviceCh chan<- []sdkModel.DiscoveredDevice) error {
	d.Logger = lc
	d.AsyncCh = asyncCh
	d.serviceConfig = &ServiceConfig{}

	ds := service.RunningService()

	if err := ds.LoadCustomConfig(d.serviceConfig, CustomConfigSectionName); err != nil {
		return fmt.Errorf("unable to load '%s' custom configuration: %s", CustomConfigSectionName, err.Error())
	}

	lc.Debugf("Custom config is: %v", d.serviceConfig)

	if err := d.serviceConfig.MQTTBrokerInfo.Validate(); err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}

	if err := ds.ListenForCustomConfigChanges(
		&d.serviceConfig.MQTTBrokerInfo.Writable,
		WritableInfoSectionName, d.updateWritableConfig); err != nil {
		return errors.NewCommonEdgeX(errors.Kind(err), fmt.Sprintf("unable to listen for changes for '%s' custom configuration", WritableInfoSectionName), err)
	}

	client, err := createMqttClient(d.serviceConfig)
	if err != nil {
		return errors.NewCommonEdgeX(errors.Kind(err), "unable to initial the MQTT client", err)
	}
	d.mqttClient = client

	return nil
}

func (d *Driver) updateWritableConfig(rawWritableConfig interface{}) {
	updated, ok := rawWritableConfig.(*WritableInfo)
	if !ok {
		d.Logger.Error("unable to update writable config: Can not cast raw config to type 'WritableInfo'")
		return
	}
	d.serviceConfig.MQTTBrokerInfo.Writable = *updated
}

func (d *Driver) DisconnectDevice(deviceName string, protocols map[string]models.ProtocolProperties) error {
	d.Logger.Warn("Driver's DisconnectDevice function didn't implement")
	return nil
}

func (d *Driver) HandleReadCommands(deviceName string, protocols map[string]models.ProtocolProperties, reqs []sdkModel.CommandRequest) ([]*sdkModel.CommandValue, error) {
	protocol := "Modbus"
	serviceName := "modbus_ds"
	topic := fmt.Sprintf("xrt/device/%s/%s/request", protocol, serviceName)
	request := xrtModel.NewDeviceResourceGetRequest(deviceName, reqs)

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, errors.NewCommonEdgeXWrapper(err)
	}

	token := d.mqttClient.Publish(topic, byte(d.serviceConfig.MQTTBrokerInfo.Qos), false, jsonData)
	if token.Wait() && token.Error() != nil {
		return nil, errors.NewCommonEdgeXWrapper(err)
	}

	cmdResponse, ok := driver.fetchCommandResponse(request.RequestId)
	if !ok {
		return nil, errors.NewCommonEdgeX(errors.KindServerError, "can not fetch command response for getting the resource", nil)
	}

	var res xrtModel.EventResponse
	err = json.Unmarshal([]byte(cmdResponse), &res)
	if err != nil {
		return nil, errors.NewCommonEdgeXWrapper(err)
	}
	if !res.Result.Success {
		return nil, errors.NewCommonEdgeX(errors.KindServerError, res.Result.Error, nil)
	}

	responses, err := commandValues(res.Result.Readings)
	if err != nil {
		return nil, errors.NewCommonEdgeXWrapper(err)
	}

	driver.Logger.Debugf("Read command response: %v", res)
	return responses, nil
}

func commandValues(readings map[string]xrtModel.Reading) ([]*sdkModel.CommandValue, errors.EdgeX) {
	var responses = make([]*sdkModel.CommandValue, len(readings))
	index := 0
	for resourceName, reading := range readings {
		valueType, err := common.NormalizeValueType(reading.Type)
		if err != nil {
			return nil, errors.NewCommonEdgeXWrapper(err)
		}
		res, err := newResult(resourceName, valueType, reading.Value)
		if err != nil {
			return nil, errors.NewCommonEdgeXWrapper(err)
		}
		responses[index] = res
		index++
	}
	return responses, nil
}

func (d *Driver) HandleWriteCommands(deviceName string, protocols map[string]models.ProtocolProperties, reqs []sdkModel.CommandRequest, params []*sdkModel.CommandValue) error {
	protocol := "Modbus"
	serviceName := "modbus_ds"
	topic := fmt.Sprintf("xrt/device/%s/%s/request", protocol, serviceName)
	request := xrtModel.NewDeviceResourceSetRequest(deviceName, reqs, params)

	jsonData, err := json.Marshal(request)
	if err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}

	token := d.mqttClient.Publish(topic, byte(d.serviceConfig.MQTTBrokerInfo.Qos), false, jsonData)
	if token.Wait() && token.Error() != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}

	cmdResponse, ok := driver.fetchCommandResponse(request.RequestId)
	if !ok {
		return errors.NewCommonEdgeX(errors.KindServerError, "can not fetch command response for writing the resources", nil)
	}

	var res xrtModel.CommonResponse
	err = json.Unmarshal([]byte(cmdResponse), &res)
	if err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}
	if !res.Result.Success {
		return errors.NewCommonEdgeX(errors.KindServerError, res.Result.Error, nil)
	}

	driver.Logger.Debugf("Write command response: %v", cmdResponse)
	return nil
}

func (d *Driver) Stop(force bool) error {
	d.Logger.Info("driver is stopping, disconnect the MQTT conn")
	if d.mqttClient.IsConnected() {
		d.mqttClient.Disconnect(5000)
	}
	return nil
}

func newResult(resourceName string, valueType string, reading interface{}) (*sdkModel.CommandValue, error) {
	var err error
	var result = &sdkModel.CommandValue{}
	castError := "fail to parse %v reading, %v"

	if !checkValueInRange(valueType, reading) {
		err = fmt.Errorf("parse reading fail. Reading %v is out of the value type(%v)'s range", reading, valueType)
		driver.Logger.Error(err.Error())
		return result, err
	}

	var val interface{}
	switch valueType {
	case common.ValueTypeBool:
		val, err = cast.ToBoolE(reading)
		if err != nil {
			return nil, fmt.Errorf(castError, resourceName, err)
		}
	case common.ValueTypeString:
		val, err = cast.ToStringE(reading)
		if err != nil {
			return nil, fmt.Errorf(castError, resourceName, err)
		}
	case common.ValueTypeUint8:
		val, err = cast.ToUint8E(reading)
		if err != nil {
			return nil, fmt.Errorf(castError, resourceName, err)
		}
	case common.ValueTypeUint16:
		val, err = cast.ToUint16E(reading)
		if err != nil {
			return nil, fmt.Errorf(castError, resourceName, err)
		}
	case common.ValueTypeUint32:
		val, err = cast.ToUint32E(reading)
		if err != nil {
			return nil, fmt.Errorf(castError, resourceName, err)
		}
	case common.ValueTypeUint64:
		val, err = cast.ToUint64E(reading)
		if err != nil {
			return nil, fmt.Errorf(castError, resourceName, err)
		}
	case common.ValueTypeInt8:
		val, err = cast.ToInt8E(reading)
		if err != nil {
			return nil, fmt.Errorf(castError, resourceName, err)
		}
	case common.ValueTypeInt16:
		val, err = cast.ToInt16E(reading)
		if err != nil {
			return nil, fmt.Errorf(castError, resourceName, err)
		}
	case common.ValueTypeInt32:
		val, err = cast.ToInt32E(reading)
		if err != nil {
			return nil, fmt.Errorf(castError, resourceName, err)
		}
	case common.ValueTypeInt64:
		val, err = cast.ToInt64E(reading)
		if err != nil {
			return nil, fmt.Errorf(castError, resourceName, err)
		}
	case common.ValueTypeFloat32:
		val, err = cast.ToFloat32E(reading)
		if err != nil {
			return nil, fmt.Errorf(castError, resourceName, err)
		}
	case common.ValueTypeFloat64:
		val, err = cast.ToFloat64E(reading)
		if err != nil {
			return nil, fmt.Errorf(castError, resourceName, err)
		}
	default:
		return nil, fmt.Errorf("return result fail, none supported value type: %v", valueType)

	}

	result, err = sdkModel.NewCommandValue(resourceName, valueType, val)
	if err != nil {
		return nil, err
	}
	result.Origin = time.Now().UnixNano()

	return result, nil
}

func newCommandValue(valueType string, param *sdkModel.CommandValue) (interface{}, error) {
	var commandValue interface{}
	var err error
	switch valueType {
	case common.ValueTypeBool:
		commandValue, err = param.BoolValue()
	case common.ValueTypeString:
		commandValue, err = param.StringValue()
	case common.ValueTypeUint8:
		commandValue, err = param.Uint8Value()
	case common.ValueTypeUint16:
		commandValue, err = param.Uint16Value()
	case common.ValueTypeUint32:
		commandValue, err = param.Uint32Value()
	case common.ValueTypeUint64:
		commandValue, err = param.Uint64Value()
	case common.ValueTypeInt8:
		commandValue, err = param.Int8Value()
	case common.ValueTypeInt16:
		commandValue, err = param.Int16Value()
	case common.ValueTypeInt32:
		commandValue, err = param.Int32Value()
	case common.ValueTypeInt64:
		commandValue, err = param.Int64Value()
	case common.ValueTypeFloat32:
		commandValue, err = param.Float32Value()
	case common.ValueTypeFloat64:
		commandValue, err = param.Float64Value()
	default:
		err = fmt.Errorf("fail to convert param, none supported value type: %v", valueType)
	}

	return commandValue, err
}

// fetchCommandResponse use to wait and fetch response from CommandResponses map
func (d *Driver) fetchCommandResponse(cmdUuid string) (string, bool) {
	var cmdResponse interface{}
	var ok bool
	for i := 0; i < 5; i++ {
		cmdResponse, ok = d.CommandResponses.Load(cmdUuid)
		if ok {
			d.CommandResponses.Delete(cmdUuid)
			break
		} else {
			time.Sleep(time.Millisecond * time.Duration(d.serviceConfig.MQTTBrokerInfo.Writable.ResponseFetchInterval))
		}
	}

	return fmt.Sprintf("%v", cmdResponse), ok
}

// fetchXRTResponse use to wait and fetch response from CommandResponses map
func (d *Driver) fetchXRTResponse(cmdUuid string, responseWaitMillisecond int) (string, bool) {
	var cmdResponse interface{}
	var ok bool
	for i := 0; i < 5; i++ {
		cmdResponse, ok = d.CommandResponses.Load(cmdUuid)
		if ok {
			d.CommandResponses.Delete(cmdUuid)
			break
		} else {
			time.Sleep(time.Millisecond * time.Duration(responseWaitMillisecond))
		}
	}

	return fmt.Sprintf("%v", cmdResponse), ok
}

func (d *Driver) AddDevice(deviceName string, protocols map[string]models.ProtocolProperties, adminState models.AdminState) error {
	d.Logger.Debugf("Device %s is added", deviceName)

	err := AddDeviceToXRT(driver.mqttClient, deviceName, protocols)
	if err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}

	return nil
}

func (d *Driver) UpdateDevice(deviceName string, protocols map[string]models.ProtocolProperties, adminState models.AdminState) error {
	d.Logger.Debugf("Device %s is updated", deviceName)
	return nil
}

func (d *Driver) RemoveDevice(deviceName string, protocols map[string]models.ProtocolProperties) error {
	d.Logger.Debugf("Device %s is removed", deviceName)
	return nil
}

func createMqttClient(serviceConfig *ServiceConfig) (mqtt.Client, errors.EdgeX) {
	var scheme = serviceConfig.MQTTBrokerInfo.Schema
	var brokerUrl = serviceConfig.MQTTBrokerInfo.Host
	var brokerPort = serviceConfig.MQTTBrokerInfo.Port
	var authMode = serviceConfig.MQTTBrokerInfo.AuthMode
	var secretPath = serviceConfig.MQTTBrokerInfo.CredentialsPath
	var mqttClientId = serviceConfig.MQTTBrokerInfo.ClientId
	var keepAlive = serviceConfig.MQTTBrokerInfo.KeepAlive

	uri := &url.URL{
		Scheme: strings.ToLower(scheme),
		Host:   fmt.Sprintf("%s:%d", brokerUrl, brokerPort),
	}

	err := SetCredentials(uri, "init", authMode, secretPath)
	if err != nil {
		return nil, errors.NewCommonEdgeXWrapper(err)
	}

	var client mqtt.Client
	for i := 0; i <= serviceConfig.MQTTBrokerInfo.ConnEstablishingRetry; i++ {
		client, err = mqttClient(mqttClientId, uri, keepAlive)
		if err != nil && i >= serviceConfig.MQTTBrokerInfo.ConnEstablishingRetry {
			return nil, errors.NewCommonEdgeXWrapper(err)
		} else if err != nil {
			driver.Logger.Warnf("Unable to connect to MQTT broker, %s, retrying", err)
			time.Sleep(time.Duration(serviceConfig.MQTTBrokerInfo.ConnEstablishingRetry) * time.Second)
			continue
		}
		break
	}
	return client, nil
}

func mqttClient(clientID string, uri *url.URL, keepAlive int) (mqtt.Client, error) {
	driver.Logger.Infof("Create MQTT client and connection: uri=%v clientID=%v ", uri.String(), clientID)
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("%s://%s", uri.Scheme, uri.Host))
	opts.SetClientID(clientID)
	opts.SetUsername(uri.User.Username())
	password, _ := uri.User.Password()
	opts.SetPassword(password)
	opts.SetKeepAlive(time.Second * time.Duration(keepAlive))
	opts.SetAutoReconnect(true)
	opts.OnConnect = onConnectHandler

	client := mqtt.NewClient(opts)
	token := client.Connect()
	if token.Wait() && token.Error() != nil {
		return client, token.Error()
	}

	return client, nil
}

func onConnectHandler(client mqtt.Client) {
	qos := byte(driver.serviceConfig.MQTTBrokerInfo.Qos)
	responseTopic := driver.serviceConfig.MQTTBrokerInfo.ResponseTopic
	incomingTopic := driver.serviceConfig.MQTTBrokerInfo.IncomingTopic

	token := client.Subscribe(incomingTopic, qos, onIncomingDataReceived)
	if token.Wait() && token.Error() != nil {
		client.Disconnect(0)
		driver.Logger.Errorf("could not subscribe to topic '%s': %s",
			incomingTopic, token.Error().Error())
		return
	}
	driver.Logger.Infof("Subscribed to topic '%s' for receiving the async reading", incomingTopic)

	token = client.Subscribe(responseTopic, qos, onCommandResponseReceived)
	if token.Wait() && token.Error() != nil {
		client.Disconnect(0)
		driver.Logger.Errorf("could not subscribe to topic '%s': %s",
			responseTopic, token.Error().Error())
		return
	}
	driver.Logger.Infof("Subscribed to topic '%s' for receiving the request response", responseTopic)

}

// Discover triggers protocol specific device discovery, which is an asynchronous operation.
func (d *Driver) Discover() {
	driver.Logger.Infof("Trigger Discover...")
	ds := service.RunningService()

	protocol := "BACnet-IP"
	serviceName := "bacnet_ds"
	bacnetTriggerTopic := fmt.Sprintf("xrt/discovery/%s/%s/request", protocol, serviceName)

	// Trigger discovery
	request := xrtModel.NewXRTRequest(xrtModel.DiscoveryTriggerOperation)
	var res xrtModel.CommonResponse
	err := d.sendXRTRequest(bacnetTriggerTopic, request, request.RequestId, 1000, &res)
	if err != nil {
		driver.Logger.Errorf("Fail to trigger discovery: %v", err)
		return
	}
	if !res.Result.Success {
		driver.Logger.Errorf("Fail to trigger discovery: %s", res.Result.Error)
		return
	}
	driver.Logger.Debug("Discovery triggered ...")

	// Check and add profile
	profiles, err := d.queryProfileListFromXRT()
	if err != nil {
		driver.Logger.Errorf("Fail to query profile list: %v", err)
		return
	}
	for _, profile := range profiles {
		_, err := ds.GetProfileByName(profile)
		if err != nil {
			p, edgexErr := d.queryProfileFromXRT(profile)
			if edgexErr != nil {
				driver.Logger.Errorf("Fail to query profile: %v", edgexErr)
				return
			}
			driver.Logger.Infof(p.Name)
			_, err := ds.AddDeviceProfile(xrtModel.Profile(p))
			if err != nil {
				driver.Logger.Errorf("Fail to add profile: %v", err)
				return
			}
		}
	}

	// Check and add device
	devices, err := d.queryDeviceListFromXRT()
	if err != nil {
		driver.Logger.Errorf("Fail to query device list: %v", err)
		return
	}
	for _, device := range devices {
		_, err := ds.GetDeviceByName(device)
		if err != nil {
			d, edgexErr := d.queryDeviceFromXRT(device)
			if edgexErr != nil {
				driver.Logger.Errorf("Fail to query device: %v", edgexErr)
				return
			}
			driver.Logger.Infof(d.Name)
			_, err := ds.AddDevice(xrtModel.Device(d))
			if err != nil {
				driver.Logger.Errorf("Fail to add device: %v", err)
				return
			}
		}
	}

}

func (d *Driver) queryProfileListFromXRT() ([]string, errors.EdgeX) {
	protocol := "BACnet-IP"
	serviceName := "bacnet_ds"
	topic := fmt.Sprintf("xrt/profile/%s/%s/request", protocol, serviceName)
	request := xrtModel.NewXRTRequest(xrtModel.ProfileListOperation)
	var profileRes xrtModel.ProfileListResponse

	err := d.sendXRTRequest(topic, request, request.RequestId, 1000, &profileRes)
	if err != nil {
		return nil, errors.NewCommonEdgeX(errors.KindServerError, fmt.Sprintf("Fail to query profile list: %v", err), nil)
	}
	if !profileRes.Result.Success {
		return nil, errors.NewCommonEdgeX(errors.KindServerError, fmt.Sprintf("Fail to query profile list: %s", profileRes.Result.Error), nil)
	}

	driver.Logger.Debugf("Profile list %v", profileRes.Result.Profiles)
	return profileRes.Result.Profiles, nil
}

func (d *Driver) queryProfileFromXRT(profileName string) (xrtModel.DeviceProfile, errors.EdgeX) {
	protocol := "BACnet-IP"
	serviceName := "bacnet_ds"
	topic := fmt.Sprintf("xrt/profile/%s/%s/request", protocol, serviceName)
	request := xrtModel.NewProfileGetRequest(profileName)
	var response xrtModel.ProfileGetResponse

	err := d.sendXRTRequest(topic, request, request.RequestId, 1000, &response)
	if err != nil {
		return xrtModel.DeviceProfile{}, errors.NewCommonEdgeX(errors.KindServerError, fmt.Sprintf("Fail to query profile: %v", err), nil)
	}
	if !response.Result.Success {
		return xrtModel.DeviceProfile{}, errors.NewCommonEdgeX(errors.KindServerError, fmt.Sprintf("Fail to query profile: %s", response.Result.Error), nil)
	}

	driver.Logger.Debugf("Profile %v", response.Result.Profile.Name)
	return response.Result.Profile, nil
}

func (d *Driver) queryDeviceListFromXRT() ([]string, errors.EdgeX) {
	protocol := "BACnet-IP"
	serviceName := "bacnet_ds"
	topic := fmt.Sprintf("xrt/device/%s/%s/request", protocol, serviceName)
	request := xrtModel.NewXRTRequest(xrtModel.DeviceListOperation)
	var response xrtModel.DeviceResponse

	err := d.sendXRTRequest(topic, request, request.RequestId, 1000, &response)
	if err != nil {
		return nil, errors.NewCommonEdgeX(errors.KindServerError, fmt.Sprintf("Fail to query device list: %v", err), nil)
	}
	if !response.Result.Success {
		return nil, errors.NewCommonEdgeX(errors.KindServerError, fmt.Sprintf("Fail to query device list: %s", response.Result.Error), nil)
	}

	driver.Logger.Debugf("Device list %v", response.Result.Devices)
	return response.Result.Devices, nil
}

func (d *Driver) queryDeviceFromXRT(deviceName string) (xrtModel.DeviceInfo, errors.EdgeX) {
	protocol := "BACnet-IP"
	serviceName := "bacnet_ds"
	topic := fmt.Sprintf("xrt/device/%s/%s/request", protocol, serviceName)
	request := xrtModel.NewDeviceGetRequest(deviceName)
	var response xrtModel.DeviceGetResponse

	err := d.sendXRTRequest(topic, request, request.RequestId, 2500, &response)
	if err != nil {
		return xrtModel.DeviceInfo{}, errors.NewCommonEdgeX(errors.KindServerError, fmt.Sprintf("Fail to query device: %v", err), nil)
	}
	if !response.Result.Success {
		return xrtModel.DeviceInfo{}, errors.NewCommonEdgeX(errors.KindServerError, fmt.Sprintf("Fail to query device: %s", response.Result.Error), nil)
	}

	driver.Logger.Debugf("Device %v", response.Result.Device.Name)
	return response.Result.Device, nil
}

func (d *Driver) sendXRTRequest(topic string, request interface{}, requestId string, responseWaitMillisecond int, response interface{}) errors.EdgeX {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}

	token := d.mqttClient.Publish(topic, byte(0), false, jsonData)
	if token.Wait() && token.Error() != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}

	cmdResponse, ok := driver.fetchXRTResponse(requestId, responseWaitMillisecond)
	if !ok {
		return errors.NewCommonEdgeX(errors.KindServerError, "Fail to fetch command response for device list", nil)
	}

	err = json.Unmarshal([]byte(cmdResponse), response)
	if err != nil {
		return errors.NewCommonEdgeX(errors.KindServerError, fmt.Sprintf("Fail to parse command response: %v", err), nil)
	}
	return nil
}
