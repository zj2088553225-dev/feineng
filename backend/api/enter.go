package api

import (
	"backend/api/service_api"
	"backend/api/user_api"
)

type ApiGroup struct {
	UserApi    user_api.UserApi
	ServiceApi service_api.ServiceApi
}

var ApiGroupApp = new(ApiGroup)
