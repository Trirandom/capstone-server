package steamhandler

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/doctype/steam"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"

	jwt "github.com/appleboy/gin-jwt"
)

type SteamSessions struct {
	Sessions map[string]*steam.Session
}

type steamFriend struct {
	ID   string `json:"id" binding:"required"`
	Name string `json:"name" binding:"required"`
}

func (s *SteamSessions) GetFriends(c *gin.Context) {
	claims := jwt.ExtractClaims(c)
	currentSession := s.Sessions[claims["id"].(string)]
	friends, err := currentSession.GetFriends(currentSession.GetSteamID())
	if err != nil {
		fmt.Print(friends)
		c.AbortWithError(http.StatusInternalServerError, errors.Wrap(err, "Get friends failed :'( "))
		return
	}

	returnFriends := []steamFriend{}
	friendSteamIDs := []string{}
	for _, friend := range friends {
		id := steam.SteamID(friend.SteamID)
		friendSteamIDs = append(friendSteamIDs, id.ToString())
	}
	fmt.Print("friend List: ", strings.Join(friendSteamIDs, ","))
	playerSummary, err := currentSession.GetPlayerSummaries(strings.Join(friendSteamIDs, ","))
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
	} else {
		for i, summary := range playerSummary {
			log.Printf("Friend Summary: %#v", summary.PersonaName)
			currentFriend := steamFriend{
				ID:   friendSteamIDs[i],
				Name: summary.PersonaName,
			}
			returnFriends = append(returnFriends, currentFriend)
		}
	}
	log.Printf("Friend Summary: %#v", returnFriends)
	c.JSON(http.StatusOK, gin.H{
		"status":  http.StatusOK,
		"friends": returnFriends,
	})
	return
}

func (s *SteamSessions) GetOwnedGames(c *gin.Context) {
	claims := jwt.ExtractClaims(c)
	currentSession := s.Sessions[claims["id"].(string)]
	ownedGames, err := currentSession.GetOwnedGames(currentSession.GetSteamID(), false, true)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, errors.Wrap(err, "Get games failed"))
		return
	}

	log.Printf("Games count: %d\n", ownedGames.Count)
	myGameIds := make([]uint32, 0)
	for _, game := range ownedGames.Games {
		// log.Printf("Game: %d 2 weeks play time: %d\n", game.AppID, game.Playtime2Weeks)
		myGameIds = append(myGameIds, game.AppID)
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  http.StatusOK,
		"gameIds": myGameIds,
	})
	return
}

type inventoryItem struct {
	ID   uint64 `json:"id" binding:"required"`
	Name string `json:"name" binding:"required"`
}

func (s *SteamSessions) GetInventory(c *gin.Context) {
	claims := jwt.ExtractClaims(c)
	currentSession := s.Sessions[claims["id"].(string)]
	sid := currentSession.GetSteamID()
	apps, err := currentSession.GetInventoryAppStats(sid)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, errors.Wrap(err, "Get inventory App stats failed"))
		return
	}

	returnItems := []inventoryItem{}
	for _, v := range apps {
		log.Printf("-- AppID total asset count: %d\n", v.AssetCount)
		for _, context := range v.Contexts {
			log.Printf("-- Items on %d %d (count %d)\n", v.AppID, context.ID, context.AssetCount)
			inventory, err := currentSession.GetInventory(sid, v.AppID, context.ID, true)
			if err != nil {
				c.AbortWithError(http.StatusInternalServerError, errors.Wrap(err, "Get inventory failed"))
				return
			}

			for _, item := range inventory {
				log.Printf("Item: %s = %d\n", item.Desc.MarketHashName, item.AssetID)
				currentItem := inventoryItem{
					ID:   item.AssetID,
					Name: item.Desc.MarketHashName,
				}
				returnItems = append(returnItems, currentItem)
			}
		}
	}

	log.Printf("Items Summary: %#v", returnItems)
	c.JSON(http.StatusOK, gin.H{
		"status": http.StatusOK,
		"items":  returnItems,
	})
	return
}

type steamCredential struct {
	Account      string `json:"account" binding:"required"`
	Password     string `json:"password" binding:"required"`
	SharedSecret string `json:"sharedSecret" binding:"omitempty"`
}

func (s *SteamSessions) SteamConnect(c *gin.Context) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	timeTip, err := steam.GetTimeTip()
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, errors.Wrap(err, "GetTimeTip() failed"))
		return
	}
	log.Printf("Time tip: %#v\n", timeTip)
	timeDiff := time.Duration(timeTip.Time - time.Now().Unix())

	// Get optional credential
	var creds steamCredential
	if err := c.ShouldBind(&creds); err != nil {
		log.Print("Binding failed ", err)
		creds = steamCredential{
			Account:  os.Getenv("steamAccount"),
			Password: os.Getenv("steamPassword"),
		}
	}

	if os.Getenv("steamApiId") == "" {
		c.AbortWithError(http.StatusInternalServerError, errors.New("Need a steam api key to login"))
		return
	}
	log.Printf("Try to log with %s %s", creds.Account, os.Getenv("steamApiId"))
	Session := steam.NewSession(&http.Client{}, os.Getenv("steamApiId"))
	if err := Session.Login(creds.Account, creds.Password, os.Getenv("steamSharedSecret"), timeDiff); err != nil {
		c.AbortWithError(http.StatusInternalServerError, errors.Wrap(err, "Login with account "+creds.Account+" failed"))
		return
	}
	claims := jwt.ExtractClaims(c)

	s.Sessions[claims["id"].(string)] = Session
	log.Print("Login with account " + creds.Account + " successful")

	c.JSON(http.StatusOK, gin.H{
		"status":     http.StatusOK,
		"message":    "Login successful with " + creds.Account,
		"resourceId": claims["id"]})
	return
}

func (s *SteamSessions) SteamAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := jwt.ExtractClaims(c)

		if s.Sessions[claims["id"].(string)] == nil || s.Sessions == nil {
			c.AbortWithStatusJSON(401, gin.H{"error": "Need to open a Steam session first with auth/steam-connexion."})
			return
		}
		c.Next()
	}
}
