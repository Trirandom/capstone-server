package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Trirandom/capstone-server/pkg/apitoolbox"
	"github.com/Trirandom/capstone-server/pkg/mongo"
	"github.com/Trirandom/capstone-server/pkg/steamhandler"
	"github.com/doctype/steam"

	jwt "github.com/appleboy/gin-jwt"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"
)

type login struct {
	Email    string `form:"email" json:"email" binding:"required"`
	Password string `form:"password" json:"password" binding:"required"`
}
type registerRequest struct {
	FirstName string `json:"firstname" binding:"required"`
	LastName  string `json:"lastname" binding:"required"`
	Password  string `json:"password" binding:"required"`
	Email     string `json:"email" binding:"required"`
}
type newsletterRequest struct {
	Name  string `json:"name" binding:"required"`
	Email string `json:"email" binding:"required"`
}

var identityKey = "id"
var steamApiKey = os.Getenv("STEAMAPIKEY")

func newsletterHandler(c *gin.Context) {
	var userInfo newsletterRequest
	fmt.Print("newsletterHandler: %#v \n", c)
	if err := c.ShouldBind(&userInfo); err != nil {
		fmt.Printf("Body: %s", userInfo)
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	ms, err := mongo.NewSession("mongodb:27017")
	if err != nil {
		log.Fatalln("unable to connect to mongodb")
	}
	ms.GetCollection("maillist").Insert(userInfo)
	fmt.Printf("success")
	c.JSON(http.StatusOK, gin.H{"status": http.StatusOK, "message": "added to mailing list", "resourceId": userInfo.Name})
	defer ms.Close()
}

func registerHandler(c *gin.Context) {
	var register registerRequest
	fmt.Printf("context: %p", c)
	if err := c.ShouldBind(&register); err != nil {
		fmt.Printf("Body: %s", register)
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	ms, err := mongo.NewSession("mongodb:27017")
	if err != nil {
		log.Fatalln("unable to connect to mongodb")
	}

	var row []DBUser = nil
	ms.GetCollection("user").Find(bson.M{"email": register.Email}).All(&row)
	if row != nil {
		defer ms.Close()
		c.JSON(http.StatusConflict, gin.H{"status": http.StatusConflict, "message": "Already exist", "resourceId": register.Email})
		return
	}

	user := DBUser{
		Email:     register.Email,
		Password:  apitoolbox.HashPassword(register.Password),
		FirstName: register.FirstName,
		LastName:  register.LastName,
	}
	err = ms.GetCollection("user").Insert(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": "Cannot insert into database", "resourceId": user.FirstName})
		log.Fatalln("unable to insert %v#", err)
	} else {
		fmt.Printf("success %#v", user)
		c.JSON(http.StatusCreated, gin.H{"status": http.StatusCreated, "message": "User created", "resourceId": user.FirstName})
	}
	defer ms.Close()
	return
}

func profileHandler(c *gin.Context) {
	fmt.Print("profileHandler: %#v", c)
	claims := jwt.ExtractClaims(c)
	// user, _ := c.Get(identityKey)
	email := claims["id"]
	ms, err := mongo.NewSession("mongodb:27017")
	if err != nil {
		log.Fatalln("unable to connect to mongodb")
	}
	var row []DBUser = nil
	ms.GetCollection("user").Find(bson.M{"email": email}).All(&row)
	defer ms.Close()
	if row != nil {
		user := User{
			Email:     row[0].Email,
			FirstName: row[0].FirstName,
			LastName:  row[0].LastName,
		}
		c.JSON(200, gin.H{
			"satatus": http.StatusOK,
			"user":    user,
		})
	} else {
		c.JSON(http.StatusNotFound, gin.H{"status": http.StatusNotFound, "message": "User not found", "resourceId": email})
	}
}

// User demo
type DBUser struct {
	Email     string `json:"email" bson:"email,omitempty"`
	FirstName string `json:"firstname" bson:"firstname,omitempty"`
	LastName  string `json:"lastname" bson:"lastname,omitempty"`
	Password  []byte `json:"password" bson:"password,omitempty"`
}
type User struct {
	Email     string
	FirstName string
	LastName  string
}

func main() {
	port := os.Getenv("PORT")
	r := gin.New()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:4200"},
		AllowMethods:     []string{"PUT", "PATCH", "POST", "GET", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin, X-Requested-With, Content-Type, Accept, Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		// AllowAllOrigins:  true,
		AllowOriginFunc: func(origin string) bool {
			return true //origin == "https://github.com"
		},
		MaxAge: 12 * time.Hour,
	}))
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
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
					identityKey: v.Email,
				}
			}
			return jwt.MapClaims{}
		},
		IdentityHandler: func(c *gin.Context) interface{} {
			claims := jwt.ExtractClaims(c)
			fmt.Println("IdentityHandler data %s", claims["id"].(string))
			return &User{
				Email: claims["id"].(string),
			}
		},
		Authenticator: func(c *gin.Context) (interface{}, error) {
			var loginVals login
			if err := c.ShouldBind(&loginVals); err != nil {
				return nil, jwt.ErrMissingLoginValues
			}
			email := loginVals.Email
			password := apitoolbox.HashPassword(loginVals.Password)
			ms, err := mongo.NewSession("mongodb:27017")
			if err != nil {
				log.Fatalln("unable to connect to mongodb")
			}
			var row []DBUser
			ms.GetCollection("user").Find(bson.M{"email": email}).All(&row)
			defer ms.Close()
			for _, value := range row {
				if email == value.Email && apitoolbox.ByteArrayCompare(password, value.Password) {
					return &User{
						Email:     email,
						LastName:  value.LastName,
						FirstName: value.FirstName,
					}, nil
				}
			}
			return nil, jwt.ErrFailedAuthentication
		},
		Authorizator: func(data interface{}, c *gin.Context) bool {
			// fmt.Println("Authorizator data ", data.(*User).FirstName)
			// if v, ok := data.(*User); ok && v.FirstName == "admin" {
			// 	fmt.Println("Authorizator v  %#v", v.FirstName)
			// 	return true
			// }
			// fmt.Println("Authorizator failed v  %#v", data.(*User))
			return true
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

	s := &steamhandler.SteamSessions{
		Sessions: map[string]*steam.Session{},
	}

	r.POST("/login", authMiddleware.LoginHandler)
	r.POST("/register", registerHandler)
	r.POST("/newsletter", newsletterHandler)

	r.NoRoute(authMiddleware.MiddlewareFunc(), func(c *gin.Context) {
		claims := jwt.ExtractClaims(c)
		log.Printf("NoRoute claims: %#v \n", claims)
		c.JSON(404, gin.H{"code": "PAGE_NOT_FOUND", "message": "Page not found"})
	})

	auth := r.Group("/auth")
	// Refresh time can be longer than token timeout
	auth.GET("/refresh_token", authMiddleware.RefreshHandler)
	auth.Use(authMiddleware.MiddlewareFunc())
	{
		auth.GET("/profile", profileHandler)

		auth.POST("/steam-connexion", s.SteamConnect)
		steamAuth := auth.Group("/steam")
		steamAuth.Use(s.SteamAuthMiddleware())
		{
			steamAuth.GET("/friend", s.GetFriends)
			steamAuth.GET("/inventory", s.GetInventory)
			steamAuth.GET("/game", s.GetOwnedGames)
		}
		// Google Calendar connexion
		// auth.GET("/calendar", mycalendar.InitCalendar)
		// auth.POST("/calendar/token", mycalendar.SaveToken)
		// auth.DELETE("/calendar/token", mycalendar.RevokeToken)
	}

	// mycalendar.InitCalendar()

	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal(err)
	}
}
