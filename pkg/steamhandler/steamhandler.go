package steamhandler

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/Philipp15b/go-steam"
	"github.com/Philipp15b/go-steam/protocol/steamlang"
	"github.com/gin-gonic/gin"
)

type steamConnexionRequest struct {
	UserName string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func SteamConnect(c *gin.Context) {

	var steamCo steamConnexionRequest
	fmt.Printf("context: %p", c)
	if err := c.ShouldBind(&steamCo); err != nil {
		fmt.Printf("Body: %s", steamCo)
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	fmt.Printf("success %#v", steamCo)

	myLoginInfo := new(steam.LogOnDetails)
	myLoginInfo.Username = steamCo.UserName
	myLoginInfo.Password = steamCo.Password

	client := steam.NewClient()
	client.Connect()
	fmt.Printf("\nsuccess  2  %#v\n", steamCo)
	for event := range client.Events() {
		switch e := event.(type) {
		case *steam.ConnectedEvent:
			client.Auth.LogOn(myLoginInfo)
			fmt.Printf("\nsuccess  A  %#v\n", steamCo)
		case *steam.MachineAuthUpdateEvent:
			ioutil.WriteFile("sentry", e.Hash, 0666)
			fmt.Printf("\nsuccess  B  %#v\n", steamCo)
		case *steam.LoggedOnEvent:
			client.Social.SetPersonaState(steamlang.EPersonaState_Online)
			fmt.Printf("\nsuccess  C  %#v\n", steamCo)
		case steam.FatalErrorEvent:
			log.Print(e)
			fmt.Printf("\nsuccess  D  %#v\n", steamCo)
		case error:
			log.Print(e)
		default:
			c.JSON(http.StatusAccepted, gin.H{"status": http.StatusAccepted, "message": "Whatever happend, just happend... [A] guess it's a good start", "resourceId": steamCo.UserName})
		}
	}
	c.JSON(http.StatusAccepted, gin.H{"status": http.StatusAccepted, "message": "Whatever happend, just happend... [B] guess it's a good start", "resourceId": steamCo.UserName})
}
