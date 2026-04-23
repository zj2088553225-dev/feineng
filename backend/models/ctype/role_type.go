package ctype

import "encoding/json"

type Role int

const (
	PermissionAdmin              Role = 1 // 管理员
	PermissionUser               Role = 2 // 普通合伙人
	PermissionAdminBranchCompany Role = 3 // 管理员

)

func (s Role) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

func (s Role) String() string {
	var str string
	switch s {
	case PermissionAdmin:
		str = "管理员"
	case PermissionAdminBranchCompany:
		str = "分公司管理员"
	case PermissionUser:
		str = "用户"
	default:
		str = "其他"
	}
	return str
}
