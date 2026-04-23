// common/common.go

package common

import (
	"backend/global"
	"backend/models"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// Option 通用分页查询选项
type Option struct {
	models.PageInfo                         // Page, Limit, Sort
	Debug           bool                    // 是否开启 debug
	Joins           string                  // 简单 JOIN
	Preload         []string                // 预加载
	Likes           []string                // 模糊匹配字段列表，如 ["OrderNo", "HBL", "CustomerName"]
	CustomCond      func(*gorm.DB) *gorm.DB // 自定义条件
	CustomSelect    string                  // 自定义 SELECT
}

// ComList 通用分页查询（带模糊搜索 + 所有扩展功能）
func ComList[T any](example T, option Option) (list []T, count int64, err error) {
	var DB = global.DB

	if option.Debug {
		DB = DB.Session(&gorm.Session{Logger: global.MysqlLog})
	}

	db := DB.Model(&example)

	// 0. 处理 Joins
	if option.Joins != "" {
		db = db.Joins(option.Joins)
	}

	query := db.Where(example)

	// 1. 预加载
	for _, preload := range option.Preload {
		query = query.Preload(preload)
	}

	// 2. 添加模糊搜索
	if option.Key != "" && len(option.Likes) > 0 {
		var conditions []string
		var args []interface{}

		for _, col := range option.Likes {
			conditions = append(conditions, fmt.Sprintf("%s LIKE ?", col))
			args = append(args, "%"+option.Key+"%")
		}

		// 使用括号包裹 OR 条件
		whereStr := "(" + strings.Join(conditions, " OR ") + ")"
		query = query.Where(whereStr, args...)
	}

	// 3. 自定义条件
	if option.CustomCond != nil {
		query = option.CustomCond(query)
	}

	// 4. 统计总数
	err = query.Count(&count).Error
	if err != nil {
		return nil, 0, err
	}
	if count == 0 {
		return []T{}, 0, nil
	}

	// 5. 分页和排序
	offset := (option.Page - 1) * option.Limit
	if offset < 0 {
		offset = 0
	}
	limit := option.Limit
	if limit == 0 {
		limit = -1 // 查询全部
	}

	// 6. 自定义 SELECT
	if option.CustomSelect != "" {
		query = query.Select(option.CustomSelect)
	}

	// 7. 执行查询
	err = query.
		Limit(limit).
		Offset(offset).
		Order(option.Sort).
		Find(&list).Error

	return list, count, err
}
