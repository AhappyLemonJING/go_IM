# 及时通讯系统

## 技术栈

GIN+MongoDB+redis+Websocket

## 安装MongoDB

* 创建network 取名为some-network

* 在some-network下创建docker 取名为some-monge
* 设置mongo的username和password以及端口号
* 最后用配置的信息在navicat上建立新的mongoDB连接

```shell
(base) wangzhujia@wangzhujiadeMacBook-Pro ~ % docker network create some-network

(base) wangzhujia@wangzhujiadeMacBook-Pro ~ % docker run -d --network some-network --name some-mongo \                  
-e MONGO_INITDB_ROOT_USERNAME=admin \
-e MONGO_INITDB_ROOT_PASSWORD=admin \
-p 27017:27017 \
mongo
```

## 创建表单

### user_basic

```json
{
		"account":"帐号",
		"password":"密码",
		"nickname":"昵称",
		"sex":1,  // 0未知 1男 2女
		"email":"邮箱",
		"avatar":"头像",
		"created_at":1, // 创建时间
		"updated_at":1, // 更新时间
}
```

### message_basic

```json
{
    "user_identity":"用户的唯一标识",
		"room_identity":"房间的唯一标识",
		"data":"发送的数据",
		"created_at":1, // 创建时间
		"updated_at":1, // 更新时间
}
```

### room_basic

```json
{
    "number":"房间号",
		"name":"房间名称",
		"info":"房间简介",
		"user_identity":"房间创建者的唯一标识",
		"created_at":1,
		"updated_at":1,
}
```

### user_room

```json
{
		"user_identity":"用户的唯一标识",
		"room_identity":"房间的唯一标识",
		"message_identity":"消息的唯一标识",
		"created_at":1,
		"updated_at":1,		
}
```

## 实现细节

### 用户登陆

#### 1. 配置路由

```go
r.POST("/login", service.Login)
```

#### 2. 登陆需要输入帐号密码再到数据库中匹配，先编写数据库查找方法

```go
models/user_basic.go

func GetUserBasicByAccountPassword(account, password string) (*UserBasic, error) {
	ub := new(UserBasic)
	err := Mongo.Collection(UserBasic{}.CollectionName()).
		FindOne(context.Background(), bson.D{{"account", account}, {"password", password}}).Decode(ub)
	return ub, err
}
```

#### 3. service中实现方法

* 从前端获取帐号和密码进行登录
* 密码经过md5加密之后，将密码和帐号从数据库中匹配，调用` models.GetUserBasicByAccountPassword(account, helper.GetMd5(password))`
* 调用`helper.GenerateToken(ub.Identity, ub.Email)`通过identity和邮箱实现token创建用于鉴权
* 登陆成功返回token

```go
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
```

### 发送验证码

#### 1. 配置路由

```go
r.POST("/send/code", service.SendCode)
```

#### 2. service中方法实现

* 获取前端输入的email
* 调用`models.GetUserBasicCountByEmail(email)`判断邮箱是否已经被注册
* 调用`helper.GetRand()`生成随机验证码，并发送到邮箱`helper.SendCode(email, code)`
* 将验证码放到redis缓存，便于后续注册时候的校验，且redis对于该验证码的key设置为 "TOKEN_"+email，限制时间5分钟

```go
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
```

#### 3. models.GetUserBasicCountByEmail(email)

```go
func GetUserBasicCountByEmail(email string) (int64, error) {
	return Mongo.Collection(UserBasic{}.CollectionName()).
		CountDocuments(context.Background(), bson.D{{"email", email}})
}
```

#### 4. helper

```go
// 生成验证码
func GetRand() string {
	rand.Seed(time.Now().UnixNano())
	s := ""
	for i := 0; i < 6; i++ {
		s = s + strconv.Itoa(rand.Intn(10))
	}
	return s
}

// 发送验证码
func SendCode(toUserEmal, code string) error {
	e := email.NewEmail()
	e.From = "wzj <wzj2010624@163.com>"
	e.To = []string{toUserEmal}
	e.Subject = "验证码已发送，请查收"
	e.HTML = []byte("你的验证码是：<b>" + code + "</b>")
	err := e.SendWithTLS("smtp.163.com:587", smtp.PlainAuth("", "wzj2010624@163.com", "RUSCZFDRNLMUYJZA", "smtp.163.com"), &tls.Config{InsecureSkipVerify: true, ServerName: "smtp.163.com"})
	return err
}
```

### 用户注册

#### 1. 配置路由

```go
r.POST("/register", service.Register)
```

#### 2. service中实现方法

* 前端获取帐号密码邮箱和验证码
* 调用`models.GetUserBasicCountByAccount(account)`判断该账号是否已被注册
* 根据对应的key从redis从获取验证码`models.RDB.Get(context.Background(), "TOKEN_"+email).Result()`
* 判断当前输入验证码和redis中的缓存是否一致
* 创建新的UserBasic将前端获取的数据存储，并自动获取uuid作为identity，密码存储加密之后的密码
* 调用`models.InsertOneUserBasic(ub)`插入数据到数据库表单中
* 调用`helper.GenerateToken(ub.Identity, ub.Email)`生成token表示直接登陆上用于鉴权，返回token

```go
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
```

#### 3. models.GetUserBasicCountByAccount(account)

```go
func GetUserBasicCountByAccount(account string) (int64, error) {
	return Mongo.Collection(UserBasic{}.CollectionName()).
		CountDocuments(context.Background(), bson.D{{"account", account}})
}
```

#### 4. models.InsertOneUserBasic(ub)

```go
func InsertOneUserBasic(ub *UserBasic) error {
	_, err := Mongo.Collection(UserBasic{}.CollectionName()).
		InsertOne(context.Background(), ub)
	return err
}
```

### 用户权限：查看个人信息

#### 0. 用户权限鉴别中间件

* 获取token，解析token
* 将解析结果存入ctx的"user_claims"中

```go
auth := r.Group("/u", middlewares.AuthCheck())

middlewares/auth.go

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
```

#### 1. 配置路由

```go
auth.POST("/user/detail", service.UserDetail)
```

#### 2. service中方法实现

* 通过`ctx.Get("user_claims")`获取出入ctx的token，来得到当前的用户
* 根据当前用户的identity来判断是否在数据库中`models.GetUserBasicByIdentity(uc.Identity)`
* 若能查到对应的信息，则显示当前用户的详细信息

```go
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
```

#### 3. models.GetUserBasicByIdentity(uc.Identity)

```go
func GetUserBasicByIdentity(identity string) (*UserBasic, error) {
	ub := new(UserBasic)
	err := Mongo.Collection(UserBasic{}.CollectionName()).
		FindOne(context.Background(), bson.D{{"identity", identity}}).Decode(ub)
	return ub, err
}
```

### 用户权限：查看指定用户的信息

#### 1. 配置路由

```go
auth.POST("/user/query", service.UserQuery)
```

#### 2. service中方法实现

* 获取前端传入的account来查找用户，` models.GetUserBasicByAcount(account)`
* 新建一个查找结果条目UserQueryResult，将数据库中对应的个人信息存储到条目中
* 判断是否和自己是好友关系
* 通过`ctx.MustGet("user_claims").(*helper.UserClaims)`获取当前登陆用户的信息
* 调用`models.JudgeUserIsFriend(ub.Identity, me.Identity)`通过查找到的用户的identity和自己的identity进行判断是否是好友关系
* 返回UserQueryResult

```go
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
```

#### 3. models.JudgeUserIsFriend(ub.Identity, me.Identity)

* 在UserRoom的表单中查找想要查询的用户且房间类型为1（私聊）的条目
* 遍历条目，将其解析，获取该条目的roomIdentity
* 遍历完之后获取到所有和useridentity1相关的私聊房间号码
* 通过该私聊房间号码和useridentity2（即当前登陆用户的identity）在uerroom表单中查询条目
* 如果能找到对应的条目，说明该查询用户和自己是好友关系

```go
func JudgeUserIsFriend(userIdentity1, userIdentity2 string) bool {
	cursor, err := Mongo.Collection(UserRoom{}.CollectionName()).Find(context.Background(), bson.D{{"user_identity", userIdentity1}, {"room_type", 1}})
	if err != nil {
		return false
	}
	roomIdentity := make([]string, 0)
	for cursor.Next(context.Background()) {
		ur := new(UserRoom)
		err = cursor.Decode(ur)
		if err != nil {
			return false
		}
		roomIdentity = append(roomIdentity, ur.RoomIdentity)
	}
	cnt, err := Mongo.Collection(UserRoom{}.CollectionName()).CountDocuments(context.Background(), bson.M{"user_identity": userIdentity2, "room_type": 1, "room_identity": bson.M{"$in": roomIdentity}})
	if err != nil {
		return false
	}
	if cnt > 0 {
		return true
	}

	return false
}
```

### 用户权限：添加好友

#### 1. 配置路由

```go
auth.POST("/user/add", service.UserAdd)
```

#### 2. service中方法实现

* 前端输入想要添加的帐号
* 通过输入的帐号查找到数据库中的用户`models.GetUserBasicByAcount(account)`
* `ctx.MustGet("user_claims").(*helper.UserClaims)`获取当前登陆信息即自己的相关信息
* 判断想要添加的帐号和自己是否是好友关系，通过两个identity进行判断，调用`models.JudgeUserIsFriend(ub.Identity, me.Identity)`
* 如果不是好友 ，就创建room_basic `models.InsertOneRoomBasic(rb)`
* 根据 roomIdentity以及俩个用户的identity，创建两个user_room且room_type=1的条目
* 分别将两个条目插入UserRoom表单中，`models.InsertOneUserRoom(ur1)` `models.InsertOneUserRoom(ur2)`

```go
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
```

#### 3. InsertOneRoomBasic

```go
func InsertOneRoomBasic(rb *RoomBasic) error {
	_, err := Mongo.Collection(RoomBasic{}.CollectionName()).
		InsertOne(context.Background(), rb)
	return err
}
```

#### 4. InsertOneUserRoom

```go
func InsertOneUserRoom(ur *UserRoom) error {
	_, err := Mongo.Collection(UserRoom{}.CollectionName()).
		InsertOne(context.Background(), ur)
	return err
}
```

### 用户权限：删除好友

#### 1. 配置路由

```go
auth.POST("/user/delete", service.UserDelete)
```

#### 2. service中方法实现

* 获取需要删除的用户的identity，如果是输入帐号删除，则仅需通过帐号查找identity即可
* 判断当前用户和需要删除的用户是否是好友`models.JudgeUserIsFriend(ub.Identity, me.Identity)`不是好友则无法进行删除
* 根据两个用户的identity获取它们私聊的roomIdentity  `models.GetUserRoomIdentity(ub.Identity, me.Identity)`
* 调用`models.DeleteByRoomIdentity(roomIdentity)`删除roomBasic
* 调用`models.DeleteUserRoomByRoomIdentity(roomIdentity)`删除UserRoom

```go
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
```

#### 3. GetUserRoomIdentity

* 找到有UserIdentity1的UserRoom条目，进行遍历，获取这些条目的roomIdentity
* 在UserRoom中查找相关的roomIdentity，且有userIdentity2的相关私聊条目
* 获取该条目的roomIdentity

```go
func GetUserRoomIdentity(userIdentity1, userIdentity2 string) string {
	cursor, err := Mongo.Collection(UserRoom{}.CollectionName()).Find(context.Background(), bson.D{{"user_identity", userIdentity1}, {"room_type", 1}})
	if err != nil {
		return ""
	}
	roomIdentity := make([]string, 0)
	for cursor.Next(context.Background()) {
		ur := new(UserRoom)
		err = cursor.Decode(ur)
		if err != nil {
			return ""
		}
		roomIdentity = append(roomIdentity, ur.RoomIdentity)
	}
	ur2 := new(UserRoom)
	err = Mongo.Collection(UserRoom{}.CollectionName()).FindOne(context.Background(), bson.M{"user_identity": userIdentity2, "room_type": 1, "room_identity": bson.M{"$in": roomIdentity}}).Decode(ur2)
	if err != nil {
		return ""
	}

	return ur2.RoomIdentity
}
```

#### 4. DeleteByRoomIdentity

```go
func DeleteByRoomIdentity(roomIdentity string) error {
	_, err := Mongo.Collection(RoomBasic{}.CollectionName()).DeleteOne(context.Background(), bson.M{"identity": roomIdentity})
	return err
}
```

#### 5. DeleteUserRoomByRoomIdentity

```go
func DeleteUserRoomByRoomIdentity(roomIdentity string) error {
	_, err := Mongo.Collection(UserRoom{}.CollectionName()).DeleteMany(context.Background(), bson.M{"room_identity": roomIdentity})
	return err
}
```

### 用户权限：发送接收消息

#### 1. 配置路由

```go
auth.GET("/websocket/message", service.WebsocketMessage)
```

#### 2. service中方法调用

* 引入websocket，通过upgrade建立连接conn`upgrader.Upgrade(ctx.Writer, ctx.Request, nil)`
* `ctx.MustGet("user_claims").(*helper.UserClaims)`获取当前登陆用户
* 根据当前登陆用户的identity设置websocket连接的键值对`wc[uc.Identity] = conn`
* 定义一个messagestruct条目，放置对应的message内容和room_identity
* 通过`conn.ReadJSON(ms)`把读到的前端发来的JSON信息存入messagestruct条目中
* 判断当前发送消息的用户是否在他想要发送的房间内`models.GetUserRoomByUserIdentityRoomIdentity(uc.Identity, ms.RoomIdentity)`
* 如果可以发送信息，则新建一个messageBasic的条目，存入相关信息，包括：房间号码，我的useridentity，消息的内容等
* 将该消息条目插入到数据库中`models.InsertOneMessageBasic(mb)`
* 获取特定房间的在线用户`models.GetUserRoomByRoomIdentity(ms.RoomIdentity)`
* 遍历用户，依次通过websocket给他们发送消息`cc.WriteMessage(websocket.TextMessage, []byte(ms.Message))`

```go
type MessageStruct struct {
	Message      string `json:"message"`
	RoomIdentity string `json:"room_identity"`
}


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

```

#### 3. GetUserRoomByUserIdentityRoomIdentity

```go
func GetUserRoomByUserIdentityRoomIdentity(userIdentity, roomIdentity string) (*UserRoom, error) {
	ur := new(UserRoom)
	err := Mongo.Collection(UserRoom{}.CollectionName()).
		FindOne(context.Background(), bson.D{{"user_identity", userIdentity}, {"room_identity", roomIdentity}}).Decode(ur)
	return ur, err
}
```

#### 4. InsertOneMessageBasic

```go
func InsertOneMessageBasic(mb *MessageBasic) error {
	_, err := Mongo.Collection(MessageBasic{}.CollectionName()).
		InsertOne(context.Background(), mb)
	return err
}
```

#### 5. GetUserRoomByRoomIdentity

```go
func GetUserRoomByRoomIdentity(roomIdentity string) ([]*UserRoom, error) {
	// 将查询到的内容解析到ur中
	urs := make([]*UserRoom, 0)
	cursor, err := Mongo.Collection(UserRoom{}.CollectionName()).Find(context.Background(), bson.D{{"room_identity", roomIdentity}})
	if err != nil {
		return nil, err
	}
	for cursor.Next(context.Background()) {
		ur := new(UserRoom)
		err = cursor.Decode(ur)
		if err != nil {
			return nil, err
		}
		urs = append(urs, ur)
	}
	return urs, nil
}
```

### 用户权限：查找聊天记录

#### 1. 配置路由

```go
auth.GET("/chat/list", service.ChatList)
```

#### 2. service中方法实现

* 获取想要查找聊天记录的roomIdentity
* 获取当前用户的信息`ctx.MustGet("user_claims").(*helper.UserClaims)`，判断当前用户是否属于该房间`models.GetUserRoomByUserIdentityRoomIdentity(uc.Identity, roomIdentity)`
* 设置显示的聊天记录的页码和一个显示多少条目
* 调用`models.GetMessageBasicByRoomIdentity(roomIdentity, &pageSize, &skip)`查找聊天记录

```go
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
```

#### 3. GetUserRoomByUserIdentityRoomIdentity

```go
func GetUserRoomByUserIdentityRoomIdentity(userIdentity, roomIdentity string) (*UserRoom, error) {
	ur := new(UserRoom)
	err := Mongo.Collection(UserRoom{}.CollectionName()).
		FindOne(context.Background(), bson.D{{"user_identity", userIdentity}, {"room_identity", roomIdentity}}).Decode(ur)
	return ur, err
}
```

####  4. GetMessageBasicByRoomIdentity

```go
func GetMessageBasicByRoomIdentity(roomIdentity string, limit, skip *int64) ([]*MessageBasic, error) {
	mbs := make([]*MessageBasic, 0)
	cursor, err := Mongo.Collection(MessageBasic{}.CollectionName()).Find(context.Background(), bson.M{"room_identity": roomIdentity}, &options.FindOptions{
		Limit: limit,
		Skip:  skip,
		Sort: bson.D{{
			"created_at", -1,
		}},
	})
	if err != nil {
		return nil, err
	}
	for cursor.Next(context.Background()) {
		mb := new(MessageBasic)
		err = cursor.Decode(mb)
		if err != nil {
			return nil, err
		}
		mbs = append(mbs, mb)
	}
	return mbs, nil
}

```

