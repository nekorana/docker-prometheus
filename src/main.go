package main

import (
	"bytes"
	"encoding/json"
	"github.com/astra/docker-prometheus/model"
	"github.com/astra/docker-prometheus/util"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
)

const (
	LarkUrl = "https://open.feishu.cn/open-apis/bot/v2/hook/4514b3a6-fa2a-4465-9a89-f8926e4fcc5f"
)

func AlertmanagerWebHook(c *gin.Context) {
	var notification model.Notification

	err := c.ShouldBindJSON(&notification)
	if err != nil {
		c.JSON(500, gin.H{
			"code":    500,
			"message": err.Error(),
		})

		return
	}

	larkRequest, _ := util.TransformToLarkRequest(notification)

	bytesData, _ := json.Marshal(larkRequest)
	req, err := http.NewRequest("POST", LarkUrl, bytes.NewReader(bytesData))
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		c.JSON(500, gin.H{
			"code":  500,
			"error": err.Error(),
		})

		return
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(res.Body)
	body, _ := io.ReadAll(res.Body)
	var larkResponse model.LarkResponse
	err = json.Unmarshal(body, &larkResponse)
	if err != nil {
		c.JSON(500, gin.H{
			"code":  500,
			"error": err.Error(),
		})

		return
	}

	c.JSON(200, gin.H{
		"code":    200,
		"message": "successful receive alert notification message!",
	})
}

func main() {
	r := gin.Default()
	r.POST("/webhook", AlertmanagerWebHook)
	_ = r.Run(":9094")
}
