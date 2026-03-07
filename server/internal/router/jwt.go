package router

import (
	"time"

	jwt "github.com/appleboy/gin-jwt/v3"
	"github.com/gin-gonic/gin"
	gojwt "github.com/golang-jwt/jwt/v5"
	"github.com/gwyy/WebTrailAI/server/internal/model/request"
	"github.com/gwyy/WebTrailAI/server/internal/model/response"
)

/*
*
login

	curl -X POST http://localhost:3457/login \
	  -H "Content-Type: application/json" \
	  -d '{
	    "username": "ebwaaa",
	    "password": "aaaaaa"
	  }' -v

export TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NzI1MzU2MDYsImlkIjoyLCJvcmlnX2lhdCI6MTc3MjUzNTU0NiwidXNlcm5hbWUiOiJlYndhYWEifQ.me4gB0vpdxvxLSx_miBry-VJ--0oo2QzYmGwiOfiDIo"

{"access_token":"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NzI1MzM2OTgsImlkIjoyLCJvcmlnX2lhdCI6MTc3MjUzMDA5OCwidXNlcm5hbWUiOiJlYndhYWEifQ.lOePHcsKpb114tVS2aQeiNCwj-T6vh5PrBaFbMZmyzQ",
"expires_in":3600,"refresh_token":"dfQHIXmi_I_-yaHdFQUlIb49LMTc9li2brqw1K-C-Gg=","token_type":"Bearer"}%

	curl -X POST http://localhost:3457/refresh_token \
	  -H "Content-Type: application/json" \
	  -H "Authorization: Bearer $TOKEN" \
	  -d '{
	    "refresh_token": "'"VovCfO1xNfniZcz_5_ROt_5WjbJ-Mp15-BLxEhKpLpE="'"
	  }' -v

		curl -X POST http://localhost:3457/logout \
		  -H "Authorization: Bearer $TOKEN" -v

		curl -X GET http://localhost:3457/api/list \
			-H "Authorization: Bearer $TOKEN" -v
*/
func (r *Router) initJWTParams(Authenticator func(c *gin.Context) (interface{}, error)) *jwt.GinJWTMiddleware {
	//初始化 jwt
	return &jwt.GinJWTMiddleware{
		Realm:           r.cfg.GetString("name"),           //显示给用户的 Realm 名称。
		Key:             []byte(r.cfg.GetString("secret")), //用于签名的密钥。
		Timeout:         time.Hour,                         //JWT Token 的有效期。
		MaxRefresh:      7 * 24 * time.Hour,                //刷新 Token 的有效期。
		IdentityKey:     "id",                              //用于在 Claims 中存储身份的键。
		PayloadFunc:     payloadFunc(),                     //向 Token 添加额外 Payload 数据的回调函数。
		IdentityHandler: identityHandler(),                 //从 Claims 检索身份的回调函数。
		Authenticator:   Authenticator,                     //验证用户的回调函数。返回用户数据。
		//Authorizer:      authorizer(),   //精细化判断用户权限 暂时不写
		Unauthorized:   unauthorized(),                                     //处理未授权请求的回调函数。
		LogoutResponse: logoutResponse(),                                   //处理成功登出响应的回调函数。
		TokenLookup:    "header: Authorization, query: token, cookie: jwt", //提取 Token 的来源（header, query, cookie）。
		// TokenLookup: "query:token",
		// TokenLookup: "cookie:token",
		TokenHeadName: "Bearer", //Header 名称前缀。
		TimeFunc:      time.Now, //提供当前时间的函数。
	}
}

// 后续请求时，用 jwt.ExtractClaims(c) 直接从 Token 里取出这些信息
// 写 Token 时用（登录/刷新时打包数据进去）
// 必须包含IdentityKey
// 这里 data 是 Authenticator 返回的结构体
func payloadFunc() func(data any) gojwt.MapClaims {
	return func(data any) gojwt.MapClaims {
		if v, ok := data.(*request.JwtUser); ok {
			return gojwt.MapClaims{
				"id":       v.ID,
				"username": v.Username,
			}
		}
		return gojwt.MapClaims{}
	}
}

// 读 Token 时用（每次请求验证后，从 Claims 里取出身份放进 context）
// 只在每次受保护路由的请求验证流程中调用（不是登录时调用 做权限验证前一步调用)
func identityHandler() func(c *gin.Context) any {
	return func(c *gin.Context) any {
		claims := jwt.ExtractClaims(c)
		idFloat, _ := claims["id"].(float64)
		return &request.JwtUser{
			ID:       int(idFloat),
			Username: claims["username"].(string),
		}
	}
}

// 处理所有未授权/验证失败请求的回调函数
func unauthorized() func(c *gin.Context, code int, message string) {
	return func(c *gin.Context, code int, message string) {
		response.FailWithCodeAndMessage(response.JwtExpired, message, c)
	}
}

func logoutResponse() func(c *gin.Context) {
	return func(c *gin.Context) {
		// This demonstrates that claims are now accessible during logout
		//claims := jwt.ExtractClaims(c)
		//user, exists := c.Get("id")
		//

		//记录 log
		response.OkWithMessage("Successfully logged out", c)
	}
}
