package model

type LarkRequest struct {
	MsgType string  `json:"msg_type"`
	Content Content `json:"content"`
}

type Content struct {
	Text string `json:"text"`
}

type LarkResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data Data   `json:"data"`
}

type Data struct{}
