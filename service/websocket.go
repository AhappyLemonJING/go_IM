package service

import (
	"IM/define"
	"IM/helper"
	"IM/models"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{}
var wc = make(map[string]*websocket.Conn)

func WebsocketMessage(ctx *gin.Context) {
	conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "系统异常" + err.Error(),
		})
		return
	}
	defer conn.Close()
	uc := ctx.MustGet("user_claims").(*helper.UserClaims)
	wc[uc.Identity] = conn
	for {
		ms := new(define.MessageStruct)
		err := conn.ReadJSON(ms)

		if err != nil {
			log.Println("read error:", err)
			return
		}
		// 判断用户是否属于消息体的房间
		_, err = models.GetUserRoomByUserIdentityRoomIdentity(uc.Identity, ms.RoomIdentity)
		if err != nil {
			log.Println("user or room not exit")
			return
		}
		// 保存消息
		mb := &models.MessageBasic{
			UserIdentity: uc.Identity,
			RoomIdentity: ms.RoomIdentity,
			Data:         ms.Message,
			CreatedAt:    time.Now().Unix(),
			UpdatedAt:    time.Now().Unix(),
		}
		err = models.InsertOneMessageBasic(mb)
		if err != nil {
			log.Println(err)
			return
		}
		// 获取在特定房间的在线用户
		urs, err := models.GetUserRoomByRoomIdentity(ms.RoomIdentity)
		if err != nil {
			log.Println(err)
			return
		}

		for _, ur := range urs {
			if cc, ok := wc[ur.UserIdentity]; ok {
				err = cc.WriteMessage(websocket.TextMessage, []byte(ms.Message))
				if err != nil {
					log.Println("write error:", err)
					return
				}
			}
		}

	}
}
