package user_api

import (
	"backend/global"
	"backend/models"
	"backend/models/res"
	"github.com/gin-gonic/gin"
)

// GET /cooperation_partner/list
func (UserApi) GetCooperationPartners(c *gin.Context) {
	var partners []models.CooperationPartner
	// 预加载用户信息
	if err := global.DB.Preload("User").Find(&partners).Error; err != nil {
		res.FailWithMessage("查询失败: "+err.Error(), c)
		return
	}

	// 组装返回数据（带用户姓名）
	var result []map[string]interface{}
	for _, p := range partners {
		var user models.UserModel
		global.DB.First(&user, p.UserID) // 获取用户信息
		result = append(result, map[string]interface{}{
			"id":        p.ID,
			"user_id":   p.UserID,
			"user_name": user.UserName,
			"rate":      p.Rate,
			"note":      p.Note,
		})
	}

	res.OkWithData(result, c)
}

// POST /cooperation_partner/add
type AddCooperationPartnerRequest struct {
	UserID uint    `json:"user_id" binding:"required"`
	Rate   float64 `json:"rate" binding:"required"`
	Note   string  `json:"note"`
}

func (UserApi) AddCooperationPartner(c *gin.Context) {
	var req AddCooperationPartnerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		res.FailWithMessage("参数错误: "+err.Error(), c)
		return
	}

	// 检查用户是否存在
	var user models.UserModel
	if err := global.DB.First(&user, req.UserID).Error; err != nil {
		res.FailWithMessage("用户不存在", c)
		return
	}

	// 检查是否已经是合营合伙人
	var exist models.CooperationPartner
	if err := global.DB.Where("user_id = ?", req.UserID).First(&exist).Error; err == nil {
		res.FailWithMessage("该用户已经是合营合伙人", c)
		return
	}

	partner := models.CooperationPartner{
		UserID: req.UserID,
		Rate:   req.Rate,
		Note:   req.Note,
	}

	if err := global.DB.Create(&partner).Error; err != nil {
		res.FailWithMessage("添加失败: "+err.Error(), c)
		return
	}

	res.OkWithMessage("添加成功", c)
}

// DELETE /cooperation_partner/delete
type DeleteCooperationPartnerRequest struct {
	ID uint `json:"id" binding:"required"`
}

func (UserApi) DeleteCooperationPartner(c *gin.Context) {
	var req DeleteCooperationPartnerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		res.FailWithMessage("参数错误: "+err.Error(), c)
		return
	}

	if err := global.DB.Delete(&models.CooperationPartner{}, req.ID).Error; err != nil {
		res.FailWithMessage("删除失败: "+err.Error(), c)
		return
	}

	res.OkWithMessage("删除成功", c)
}

// 请求体
type EditCooperationRequest struct {
	ID   uint    `json:"id"`   // 合营合伙人记录ID
	Rate float64 `json:"rate"` // 合营比例
	Note string  `json:"note"` // 备注
}

// PUT /cooperation_partner/edit
func (UserApi) EditCooperationPartner(c *gin.Context) {
	var req EditCooperationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		res.FailWithMessage("参数错误: "+err.Error(), c)
		return
	}

	var partner models.CooperationPartner
	if err := global.DB.First(&partner, req.ID).Error; err != nil {
		res.FailWithMessage("合营合伙人不存在", c)
		return
	}

	partner.Rate = req.Rate
	partner.Note = req.Note

	if err := global.DB.Save(&partner).Error; err != nil {
		res.FailWithMessage("更新失败: "+err.Error(), c)
		return
	}

	res.OkWithMessage("更新成功", c)
}
