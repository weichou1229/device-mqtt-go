package models

type XRTResponse struct {
	Client    string `json:"client"`
	RequestId string `json:"request_id"`
	Op        string `json:"op"`
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
