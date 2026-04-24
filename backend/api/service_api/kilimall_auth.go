package service_api

import (
	"backend/global"
	"backend/models"
	"backend/models/res"
	"bytes"
	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
	"os"
	"strings"
)

const kilimallSettingsPath = "../sync_data/settings.yaml"
const kilimallServiceStatusID = 13

type KilimallAuthUpdateRequest struct {
	Cookie string `json:"cookie" binding:"required" msg:"请输入 Kilimall Seller-SID Cookie"`
	Token  string `json:"token" binding:"required" msg:"请输入 Kilimall AccessToken"`
}

func (ServiceApi) UpdateKilimallCookieView(c *gin.Context) {
	var req KilimallAuthUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		res.FailWithError(err, &req, c)
		return
	}

	cookie := strings.TrimSpace(req.Cookie)
	token := strings.TrimSpace(req.Token)
	if cookie == "" || token == "" {
		res.FailWithMessage("Kilimall Cookie 和 AccessToken 不能为空", c)
		return
	}

	if err := updateKilimallSettings(cookie, token); err != nil {
		global.Log.Errorf("更新 Kilimall 授权配置失败: %v", err)
		res.FailWithMessage("更新 Kilimall 授权失败", c)
		return
	}

	if err := global.DB.Where("id = ?", kilimallServiceStatusID).Updates(models.ServiceStatus{
		Status:  "正常",
		Message: "Kilimall 授权已手动更新",
	}).Error; err != nil {
		global.Log.Errorf("重置 Kilimall 服务状态失败: %v", err)
		res.FailWithMessage("授权已保存，但服务状态重置失败", c)
		return
	}

	res.OkWithMessage("Kilimall 授权更新成功", c)
}

func updateKilimallSettings(cookie, token string) error {
	content, err := os.ReadFile(kilimallSettingsPath)
	if err != nil {
		return err
	}

	var root yaml.Node
	if err := yaml.Unmarshal(content, &root); err != nil {
		return err
	}
	if len(root.Content) == 0 {
		return nil
	}

	kilimall := mappingValue(root.Content[0], "kilimall")
	if kilimall == nil {
		root.Content[0].Content = append(root.Content[0].Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "kilimall"},
			&yaml.Node{Kind: yaml.MappingNode},
		)
		kilimall = root.Content[0].Content[len(root.Content[0].Content)-1]
	}

	setMappingScalar(kilimall, "cookie", cookie)
	setMappingScalar(kilimall, "auth_token", token)

	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(4)
	if err := encoder.Encode(&root); err != nil {
		return err
	}
	if err := encoder.Close(); err != nil {
		return err
	}

	return os.WriteFile(kilimallSettingsPath, buf.Bytes(), 0644)
}

func mappingValue(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

func setMappingScalar(node *yaml.Node, key, value string) {
	if node.Kind != yaml.MappingNode {
		node.Kind = yaml.MappingNode
		node.Content = nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			node.Content[i+1].Kind = yaml.ScalarNode
			node.Content[i+1].Tag = "!!str"
			node.Content[i+1].Value = value
			return
		}
	}
	node.Content = append(node.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: key},
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value},
	)
}
