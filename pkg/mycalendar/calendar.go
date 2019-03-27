package mycalendar

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
)

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		// tok = getTokenFromWeb(config)
		// saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// Request a token from the web, then returns the retrieved token.
// return *oauth2.Token
func getTokenFromWeb(config *oauth2.Config) string {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	return authURL
	// var authCode string
	// if _, err := fmt.Scan(&authCode); err != nil {
	// 	log.Fatalf("Unable to read authorization code: %v", err)
	// }

	// tok, err := config.Exchange(context.TODO(), authCode)
	// if err != nil {
	// 	log.Fatalf("Unable to retrieve token from web: %v", err)
	// }
	// return tok
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

type tokenRequest struct {
	Token string `json:"token" binding:"required"`
}

func RevokeToken(c *gin.Context) {
	path := "token.json"
	err := os.Remove(path)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		log.Fatalf("Unable to delete token: %v", err)
	}
	c.AbortWithStatus(http.StatusOK)
}

// Saves a token to a file path.
// param:   path string, token *oauth2.Token
func SaveToken(c *gin.Context) {
	var token tokenRequest
	if err := c.ShouldBind(&token); err != nil {
		fmt.Printf("Body: %s\n", token)
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	fmt.Printf("Body: %s\n", token)

	path := "token.json"
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		log.Fatalf("Unable to cache oauth token: %v\n", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func InitCalendar(c *gin.Context) {
	b, err := ioutil.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v\n", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, calendar.CalendarReadonlyScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v\n", err)
	}

	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		authURL := getTokenFromWeb(config)
		c.JSON(http.StatusCreated, gin.H{"status": http.StatusCreated, "url": authURL})
	} else {
		// client := getClient(config)
		client := config.Client(context.Background(), tok)

		srv, err := calendar.New(client)
		if err != nil {
			errMsg := "Unable to retrieve Calendar client"
			c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": errMsg})
			// log.Fatalf("%s: %v\n", errMsg, err)
		}

		t := time.Now().Format(time.RFC3339)
		events, err := srv.Events.List("primary").ShowDeleted(false).
			SingleEvents(true).TimeMin(t).MaxResults(10).OrderBy("startTime").Do()
		if err != nil {
			errMsg := "Unable to retrieve next ten of the user's events"
			c.JSON(http.StatusUnauthorized, gin.H{"status": http.StatusUnauthorized, "message": errMsg, "server": err})
			path := "token.json"
			os.Remove(path)
			// log.Fatalf("%s: %v\n", errMsg, err)
			return
		}
		fmt.Println("Upcoming events:")
		if len(events.Items) == 0 {
			c.JSON(http.StatusNoContent, gin.H{"status": http.StatusNoContent})
			fmt.Println("No upcoming events found.\n")
		} else {
			c.JSON(http.StatusOK, gin.H{"status": http.StatusOK, "events": events})
			for _, item := range events.Items {
				date := item.Start.DateTime
				if date == "" {
					date = item.Start.Date
				}
				fmt.Printf("%v (%v)\n", item.Summary, date)
			}
		}
	}
}
