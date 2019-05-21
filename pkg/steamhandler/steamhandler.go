package steamhandler

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/doctype/steam"
	"github.com/gin-gonic/gin"

	jwt "github.com/appleboy/gin-jwt"
)

type SteamSessions struct {
	Sessions map[string]*steam.Session
}

func (s *SteamSessions) GetFriends(c *gin.Context) {
	claims := jwt.ExtractClaims(c)
	currentSession := s.Sessions[claims["id"].(string)]
	friends, err := currentSession.GetFriends(currentSession.GetSteamID())
	if err != nil {
		log.Panic(err)
	}
	for _, friend := range friends {
		log.Printf("Friend: %#v", friend)
		friendStID := steam.SteamID(friend.SteamID)
		playerSummary, err := currentSession.GetPlayerSummaries(friendStID.ToString())
		if err != nil {
			log.Printf("getPlayer error: ", err)
		} else {
			for _, summary := range playerSummary {
				log.Printf("Player Summary: %#v", summary.PersonaName)
			}
		}
	}
	return
}

func (s *SteamSessions) GetOwnedGames(c *gin.Context) {
	claims := jwt.ExtractClaims(c)
	currentSession := s.Sessions[claims["id"].(string)]
	ownedGames, err := currentSession.GetOwnedGames(currentSession.GetSteamID(), false, true)
	if err != nil {
		log.Panic(err)
	}

	log.Printf("Games count: %d\n", ownedGames.Count)
	myGameIds := make([]uint32, 0)
	for _, game := range ownedGames.Games {
		log.Printf("Game: %d 2 weeks play time: %d\n", game.AppID, game.Playtime2Weeks)
		myGameIds = append(myGameIds, game.AppID)
	}

	c.JSON(200, gin.H{
		"status":     http.StatusOK,
		"myGameIds":  myGameIds,
		"resourceId": claims["id"],
	})
	return
}

func (s *SteamSessions) GetInventory(c *gin.Context) {
	claims := jwt.ExtractClaims(c)
	currentSession := s.Sessions[claims["id"].(string)]
	sid := currentSession.GetSteamID()
	apps, err := currentSession.GetInventoryAppStats(sid)
	if err != nil {
		log.Panic(err)
	}
	for _, v := range apps {
		log.Printf("-- AppID total asset count: %d\n", v.AssetCount)
		for _, context := range v.Contexts {
			log.Printf("-- Items on %d %d (count %d)\n", v.AppID, context.ID, context.AssetCount)
			inven, err := currentSession.GetInventory(sid, v.AppID, context.ID, true)
			if err != nil {
				log.Panic(err)
			}

			for _, item := range inven {
				log.Printf("Item: %s = %d\n", item.Desc.MarketHashName, item.AssetID)
			}
		}
	}
	return
}

func (s *SteamSessions) SteamConnect(c *gin.Context) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	timeTip, err := steam.GetTimeTip()
	if err != nil {
		log.Panic(err)
	}
	log.Printf("Time tip: %#v\n", timeTip)
	timeDiff := time.Duration(timeTip.Time - time.Now().Unix())

	log.Print("account %s", os.Getenv("steamAccount"))

	Session := steam.NewSession(&http.Client{}, os.Getenv("steamApiId"))
	if err := Session.Login(os.Getenv("steamAccount"), os.Getenv("steamPassword"), os.Getenv("steamSharedSecret"), timeDiff); err != nil {
		log.Panic(err)
	}
	claims := jwt.ExtractClaims(c)

	s.Sessions[claims["id"].(string)] = Session

	log.Print("Login successful")

	// sid := steam.SteamID(Session.GetSteamID())
	c.JSON(http.StatusOK, gin.H{"status": http.StatusOK, "message": "Login successful", "resourceId": claims["id"]})
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
