package models

type XRTResponse struct {
	Client    string `json:"client"`
	RequestId string `json:"request_id"`
}

type CommonResponse struct {
	XRTResponse `json:",inline"`
	Result      CommonResult `json:"result"`
}

type CommonResult struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

type EventResponse struct {
	XRTResponse `json:",inline"`
	Result      EventResult `json:"result"`
}

type EventResult struct {
	CommonResult `json:",inline"`
	Readings     map[string]Reading `json:"readings"`
}

type Reading struct {
	Value interface{} `json:"value"`
	Type  string      `json:"type"`
}

type DeviceResponse struct {
	XRTResponse `json:",inline"`
	Result      DeviceListResult `json:"result"`
}

type DeviceListResult struct {
	CommonResult `json:",inline"`
	Devices      []string `json:"devices"`
}

type DeviceGetResponse struct {
	XRTResponse `json:",inline"`
	Result      DeviceGetResult `json:"result"`
}

type DeviceGetResult struct {
	CommonResult `json:",inline"`
	Device       DeviceInfo `json:"device"`
}

type ProfileListResponse struct {
	XRTResponse `json:",inline"`
	Result      ProfileListResult `json:"result"`
}

type ProfileListResult struct {
	CommonResult `json:",inline"`
	Profiles     []string `json:"profiles"`
}

type ProfileGetResponse struct {
	CommonResult `json:",inline"`
	Result       ProfileGetResult `json:"result"`
}

type ProfileGetResult struct {
	CommonResult `json:",inline"`
	Profile      DeviceProfile `json:"profile"`
}
