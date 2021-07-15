// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2018-2019 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package driver

import (
	"encoding/json"

	"github.com/eclipse/paho.mqtt.golang"
)

func onCommandResponseReceived(client mqtt.Client, message mqtt.Message) {
	var response map[string]interface{}

	json.Unmarshal(message.Payload(), &response)
	requestId, ok := response["request_id"].(string)
	if ok {
		driver.CommandResponses.Store(requestId, string(message.Payload()))
		//driver.Logger.Debugf("[Response listener] Command response received: topic=%v requestId=%v msg=%v", message.Topic(), requestId, string(message.Payload()))
	} else {
		driver.Logger.Warnf("[Response listener] Command response ignored. No requestId found in the message: topic=%v msg=%v", message.Topic(), string(message.Payload()))
	}
}
