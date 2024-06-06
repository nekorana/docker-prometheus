package util

import (
	"bytes"
	"fmt"
	"github.com/astra/docker-prometheus/model"
)

func TransformToLarkRequest(notification model.Notification) (larkRequest *model.LarkRequest, err error) {
	var buffer bytes.Buffer

	buffer.WriteString(fmt.Sprintf("通知组%s, 状态[%s]\n告警项\n\n", notification.GroupKey, notification.Status))

	for _, alert := range notification.Alerts {
		buffer.WriteString(fmt.Sprintf("摘要: %s\n\n详情: %s\n", alert.Annotations["summary"], alert.Annotations["description"]))
		buffer.WriteString(fmt.Sprintf("开始时间: %s\n\n", alert.StartsAt.Format("15:04:05")))
	}

	larkRequest = &model.LarkRequest{
		MsgType: "text",
		Content: model.Content{
			Text: buffer.String(),
		},
	}

	return larkRequest, nil
}