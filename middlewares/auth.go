package middlewares

import (
	"IM/helper"
	"net/http"

	"github.com/gin-gonic/gin"
)

func AuthCheck() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		token := ctx.GetHeader("token")
		userClaims, err := helper.AnalyseToken(token)
		if err != nil {
			ctx.Abort()
			ctx.JSON(http.StatusOK, gin.H{
				"code": -1,
				"msg":  "用户认证不通过",
			})
			return
		}
		ctx.Set("user_claims", userClaims)
		ctx.Next()
	}
}
