package test

import (
	"flag"
	"log"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// websocket包下面的demo可以直接拷贝过来用

var addr = flag.String("addr", "127.0.0.1:8085", "http service address")
var upgrader = websocket.Upgrader{}
var ws = make(map[*websocket.Conn]struct{})

func echo(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil) //c是一个connection
	if err != nil {
		log.Println("upgrade:", err)
		return
	}
	defer c.Close()
	ws[c] = struct{}{}
	for {
		mt, message, err := c.ReadMessage() // 读数据
		if err != nil {
			log.Println("read:", err)
			break
		}
		log.Printf("recv:%s", message)
		for conn := range ws {
			err = conn.WriteMessage(mt, message) // 往我们的端发数据
			if err != nil {
				log.Println("write:", err)
				break
			}
		}

	}
}
func TestWebsocketServer(t *testing.T) {
	http.HandleFunc("/echo", echo)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

func TestGinWebsocketServer(t *testing.T) {
	r := gin.Default()
	r.GET("/echo", func(ctx *gin.Context) {
		echo(ctx.Writer, ctx.Request)
	})
	r.Run(":8085")

}
