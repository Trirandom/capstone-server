package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Trirandom/capstone/server/pkg/mongo"
	jwt "github.com/appleboy/gin-jwt"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/argon2"
	"gopkg.in/mgo.v2/bson"
)

type login struct {
	Username string `form:"username" json:"username" binding:"required"`
	Password string `form:"password" json:"password" binding:"required"`
}
type registerRequest struct {
	Username string `form:"username" json:"username" binding:"required"`
	Password string `form:"password" json:"password" binding:"required"`
}

var identityKey = "id"

func helloHandler(c *gin.Context) {
	claims := jwt.ExtractClaims(c)
	user, _ := c.Get(identityKey)
	c.JSON(200, gin.H{
		"userID":   claims["id"],
		"userName": user.(*User).UserName,
		"text":     "Hello World.",
	})
}

func hashPassword(password string) []byte {
	return argon2.Key([]byte(password), []byte("mysalt8YI56780IJLKETRD4gsdrstyy'3-(é'zdhgs"), 3, 32*1024, 4, 32)
}

func registerHandler(c *gin.Context) {
	var register registerRequest
	if err := c.ShouldBind(&register); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	ms, err := mongo.NewSession("127.0.0.1:27017")
	if err != nil {
		log.Fatalln("unable to connect to mongodb")
	}
	user := DBUser{
		UserName: register.Username,
		Password: hashPassword(register.Password),
	}
	ms.GetCollection("user").Insert(user)
	fmt.Printf("success")
	defer ms.Close()
	return
}

func compare(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if b[i] != v {
			return false
		}
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

// User demo
type DBUser struct {
	UserName  string `json:"username" bson:"username,omitempty"`
	FirstName string `json:"firstname" bson:"firstname,omitempty"`
	LastName  string `json:"lastname" bson:"lastname,omitempty"`
	Password  []byte `json:"password" bson:"password,omitempty"`
}
type User struct {
	UserName  string
	FirstName string
	LastName  string
}

func main() {
	port := os.Getenv("PORT")
	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	fmt.Println("hello from main")

	if port == "" {
		port = "8080"
	}

	// the jwt middleware
	authMiddleware, err := jwt.New(&jwt.GinJWTMiddleware{
		Realm:       "test zone",
		Key:         []byte("je suis un chasseur lalalala"),
		Timeout:     time.Hour,
		MaxRefresh:  time.Hour,
		IdentityKey: identityKey,
		PayloadFunc: func(data interface{}) jwt.MapClaims {
			if v, ok := data.(*User); ok {
				return jwt.MapClaims{
					identityKey: v.UserName,
				}
			}
			return jwt.MapClaims{}
		},
		IdentityHandler: func(c *gin.Context) interface{} {
			claims := jwt.ExtractClaims(c)
			return &User{
				UserName: claims["id"].(string),
			}
		},
		Authenticator: func(c *gin.Context) (interface{}, error) {
			var loginVals login
			if err := c.ShouldBind(&loginVals); err != nil {
				return nil, jwt.ErrMissingLoginValues
			}
			userID := loginVals.Username
			password := hashPassword(loginVals.Password)
			ms, err := mongo.NewSession("127.0.0.1:27017")
			if err != nil {
				log.Fatalln("unable to connect to mongodb")
			}
			var row []DBUser
			ms.GetCollection("user").Find(bson.M{"username": userID}).All(&row)
			defer ms.Close()
			for _, value := range row {
				if userID == value.UserName && compare(password, value.Password) {
					return &User{
						UserName:  userID,
						LastName:  "BG du 67",
						FirstName: "Wow",
					}, nil
				}
			}
			return nil, jwt.ErrFailedAuthentication
		},
		Authorizator: func(data interface{}, c *gin.Context) bool {
			if v, ok := data.(*User); ok && v.UserName == "admin" {
				return true
			}

			return false
		},
		Unauthorized: func(c *gin.Context, code int, message string) {
			c.JSON(code, gin.H{
				"code":    code,
				"message": message,
			})
		},
		// TokenLookup is a string in the form of "<source>:<name>" that is used
		// to extract token from the request.
		// Optional. Default value "header:Authorization".
		// Possible values:
		// - "header:<name>"
		// - "query:<name>"
		// - "cookie:<name>"
		// - "param:<name>"
		TokenLookup: "header: Authorization, query: token, cookie: jwt",
		// TokenLookup: "query:token",
		// TokenLookup: "cookie:token",

		// TokenHeadName is a string in the header. Default value is "Bearer"
		TokenHeadName: "Bearer",

		// TimeFunc provides the current time. You can override it to use another time value. This is useful for testing or if your server uses a different time zone than your tokens.
		TimeFunc: time.Now,
	})

	if err != nil {
		log.Fatal("JWT Error:" + err.Error())
	}

	r.POST("/login", authMiddleware.LoginHandler)
	r.POST("/register", registerHandler)

	r.NoRoute(authMiddleware.MiddlewareFunc(), func(c *gin.Context) {
		claims := jwt.ExtractClaims(c)
		log.Printf("NoRoute claims: %#v\n", claims)
		c.JSON(404, gin.H{"code": "PAGE_NOT_FOUND", "message": "Page not found"})
	})

	auth := r.Group("/auth")
	// Refresh time can be longer than token timeout
	auth.GET("/refresh_token", authMiddleware.RefreshHandler)
	auth.Use(authMiddleware.MiddlewareFunc())
	{
		auth.GET("/hello", helloHandler)
	}

	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal(err)
	}
}
