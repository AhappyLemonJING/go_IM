package service

import (
	"IM/helper"
	"IM/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func ChatList(ctx *gin.Context) {
	roomIdentity := ctx.Query("room_identity")
	if roomIdentity == "" {
		ctx.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "房间号不能为空",
		})
		return
	}
	// 判断用户是否属于该房间
	uc := ctx.MustGet("user_claims").(*helper.UserClaims)

	_, err := models.GetUserRoomByUserIdentityRoomIdentity(uc.Identity, roomIdentity)
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "非法访问",
		})
		return
	}
	pageIndex, _ := strconv.ParseInt(ctx.Query("page_index"), 10, 32)
	pageSize, _ := strconv.ParseInt(ctx.Query("page_size"), 10, 32)
	skip := (pageIndex - 1) * pageSize
	// 聊天记录查找
	data, err := models.GetMessageBasicByRoomIdentity(roomIdentity, &pageSize, &skip)
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "系统异常" + err.Error(),
		})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "查询成功",
		"data": gin.H{
			"list": data,
		},
	})

}
