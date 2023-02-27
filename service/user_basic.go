package service

import (
	"IM/helper"
	"IM/models"
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type UserQueryResult struct {
	Nickname string `json:"nickname"`
	Sex      int    `json:"sex"`
	Email    string `json:"email"`
	Avatar   string `json:"avater"`
	IsFriend bool   `json:"is_friend"`
}

func Login(ctx *gin.Context) {
	account := ctx.Query("account")
	password := ctx.Query("password")
	fmt.Println(account, password)
	if account == "" || password == "" {
		ctx.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "用户名和密码不能为空",
		})
		return
	}
	fmt.Println(helper.GetMd5("wzjjj"))
	ub, err := models.GetUserBasicByAccountPassword(account, helper.GetMd5(password))
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "用户名和密码错误",
		})
		return
	}
	token, err := helper.GenerateToken(ub.Identity, ub.Email)
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "系统错误",
		})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "登陆成功",
		"data": gin.H{
			"token": token,
		},
	})
}

func UserDetail(ctx *gin.Context) {
	u, _ := ctx.Get("user_claims")
	uc := u.(*helper.UserClaims)

	ub, err := models.GetUserBasicByIdentity(uc.Identity)
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "数据查询异常",
		})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "查询成功",
		"data": ub,
	})

}

func UserQuery(ctx *gin.Context) {
	account := ctx.Query("account")
	if account == "" {
		ctx.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "查询帐号不能为空",
		})
		return
	}
	ub, err := models.GetUserBasicByAcount(account)
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "数据查询异常",
		})
		return
	}

	result := UserQueryResult{
		Nickname: ub.Nickname,
		Sex:      ub.Sex,
		Email:    ub.Email,
		Avatar:   ub.Avatar,
		IsFriend: false,
	}
	// 判断是否为好友
	// 获取当前identity
	me := ctx.MustGet("user_claims").(*helper.UserClaims)
	if models.JudgeUserIsFriend(ub.Identity, me.Identity) {
		result.IsFriend = true
	}
	ctx.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "查询成功",
		"data": result,
	})

}

func SendCode(ctx *gin.Context) {
	email := ctx.Query("email")
	if email == "" {
		ctx.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "邮箱不为空",
		})
		return
	}
	cnt, err := models.GetUserBasicCountByEmail(email)
	if err != nil {
		log.Println(err)
		ctx.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "系统错误",
		})
		return
	}
	if cnt > 1 {
		log.Println(err)
		ctx.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "邮箱已经被注册",
		})
		return
	}
	code := helper.GetRand()

	err = helper.SendCode(email, code)
	if err != nil {
		log.Println(err)
		ctx.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "系统错误",
		})
		return
	}
	err = models.RDB.Set(context.Background(), "TOKEN_"+email, code, time.Second*300).Err()
	if err != nil {
		log.Println(err)
		ctx.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "redis存储错误",
		})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "验证码发送成功",
	})

}

func Register(ctx *gin.Context) {
	account := ctx.Query("account")
	password := ctx.Query("password")
	email := ctx.Query("email")
	code := ctx.Query("code")
	if email == "" || password == "" || account == "" || code == "" {
		ctx.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "参数不正确",
		})
		return
	}
	cnt, err := models.GetUserBasicCountByAccount(account)
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "系统错误",
		})
		return
	}
	if cnt > 0 {
		ctx.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "该帐号已经被注册",
		})
		return
	}

	// 校验验证码
	needCode, err := models.RDB.Get(context.Background(), "TOKEN_"+email).Result()
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "系统错误",
		})
		return
	}
	if needCode != code {
		ctx.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "验证码错误",
		})
		return
	}
	ub := &models.UserBasic{
		Account:   account,
		Password:  helper.GetMd5(password),
		Email:     email,
		Identity:  helper.GetUUID(),
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}
	err = models.InsertOneUserBasic(ub)
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "插入失败" + err.Error(),
		})
		return
	}
	token, err := helper.GenerateToken(ub.Identity, ub.Email)
	ctx.JSON(http.StatusOK, gin.H{
		"code": -1,
		"msg":  "注册成功",
		"data": gin.H{
			"token": token,
		},
	})

}

func UserAdd(ctx *gin.Context) {
	account := ctx.Query("account")
	if account == "" {
		ctx.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "帐号不能为空",
		})
		return
	}
	// 判断该账户和自己是否是好友
	ub, err := models.GetUserBasicByAcount(account)
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "系统错误",
		})
		return
	}
	me := ctx.MustGet("user_claims").(*helper.UserClaims)
	if models.JudgeUserIsFriend(ub.Identity, me.Identity) {
		ctx.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "当前账户已经是你的好友了",
		})
		return
	}
	// 如果不是好友  就创建room_basic user_room且room_type=1
	rb := &models.RoomBasic{
		Identity:     helper.GetUUID(),
		UserIdentity: me.Identity,
		CreatedAt:    time.Now().Unix(),
		UpdatedAt:    time.Now().Unix(),
	}
	if err := models.InsertOneRoomBasic(rb); err != nil {
		ctx.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "插入roombasic失败",
		})
		return
	}
	ur1 := &models.UserRoom{
		UserIdentity: ub.Identity,
		RoomIdentity: rb.Identity,
		RoomType:     1,
		CreatedAt:    time.Now().Unix(),
		UpdatedAt:    time.Now().Unix(),
	}
	ur2 := &models.UserRoom{
		UserIdentity: me.Identity,
		RoomIdentity: rb.Identity,
		RoomType:     1,
		CreatedAt:    time.Now().Unix(),
		UpdatedAt:    time.Now().Unix(),
	}
	if err := models.InsertOneUserRoom(ur1); err != nil {
		ctx.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "插入userroom1失败",
		})
		return
	}
	if err := models.InsertOneUserRoom(ur2); err != nil {
		ctx.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "插入userroom2失败",
		})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "添加好友成功",
	})

}

func UserDelete(ctx *gin.Context) {
	identity := ctx.Query("identity")
	if identity == "" {
		ctx.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "参数错误",
		})
		return
	}
	// 判断是否为好友
	ub, err := models.GetUserBasicByIdentity(identity)
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "系统错误",
		})
		return
	}
	me := ctx.MustGet("user_claims").(*helper.UserClaims)
	if !models.JudgeUserIsFriend(ub.Identity, me.Identity) {
		ctx.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "不是好友，无法删除",
		})
		return
	}
	roomIdentity := models.GetUserRoomIdentity(ub.Identity, me.Identity)
	// 删除room_basic
	err = models.DeleteByRoomIdentity(roomIdentity)
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "删除失败",
		})
		return
	}

	// 删除user_room
	err = models.DeleteUserRoomByRoomIdentity(roomIdentity)
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "删除失败",
		})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "删除成功",
	})

}
