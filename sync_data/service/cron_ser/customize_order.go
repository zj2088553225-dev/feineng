package cron_ser

import (
	"fmt"
	"sync_data/global"
	"sync_data/models"
	"sync_data/service/fill_customize_order"
)

func SyncCustomizeOrder() {
	global.Log.Info("定时同步订单数据开始")
	_, err := fill_customize_order.FillAllCustomizeOrderKE(100)
	if err != nil {
		global.DB.Where("id = ?", 7).Updates(models.ServiceStatus{
			Status:  "错误",
			Message: fmt.Errorf("同步肯尼亚社媒数据错误：%s", err.Error()).Error(),
		})
		global.Log.Error(err.Error())
		return
	}
	_, err = fill_customize_order.FillAllCustomizeOrderGH(100)
	if err != nil {
		global.DB.Where("id = ?", 7).Updates(models.ServiceStatus{
			Status:  "错误",
			Message: fmt.Errorf("同步加纳社媒数据错误：%s", err.Error()).Error(),
		})
		global.Log.Error(err.Error())
		return
	}
	_, err = fill_customize_order.FillAllCustomizeOrderNG(100)
	if err != nil {
		global.DB.Where("id = ?", 7).Updates(models.ServiceStatus{
			Status:  "错误",
			Message: fmt.Errorf("同步尼日利亚社媒数据错误：%s", err.Error()).Error(),
		})
		global.Log.Error(err.Error())
		return
	}
	global.DB.Where("id = ?", 7).Updates(models.ServiceStatus{
		Status:  "正常",
		Message: fmt.Sprintf("定时同步社媒订单数据成功"),
	})
}
