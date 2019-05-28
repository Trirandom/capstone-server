package mongo

import (
	"os"

	mgo "gopkg.in/mgo.v2"
)

type Session struct {
	session *mgo.Session
}

const (
	mongoDBHosts = "localhost:27017" // localhost for local & mongodb for docker
	database     = "capstone"
	userName     = "capstone-user"
	password     = "les_patates_sont_cuites"
)

// NewSession create a new mongodb session
func NewSession(url string) (*Session, error) {

	// mongoDBDialInfo := &mgo.DialInfo{
	// 	Addrs:    []string{mongoDBHosts},
	// 	Timeout:  60 * time.Second,
	// 	Database: database,
	// 	Username: userName,
	// 	Password: password,
	// }
	// session, err := mgo.DialWithInfo(mongoDBDialInfo)

	var mongoURL = "localhost:27017"
	if os.Getenv("CAPSTONE_PRODUCTION") == "true" {
		mongoURL = "mongodb:27017"
	}
	session, err := mgo.Dial(mongoURL)
	if err != nil {
		return nil, err
	}
	// session.Login(&mgo.Credential{Username: userName, Password: password})
	return &Session{session}, err
}

func (s *Session) Copy() *Session {
	return &Session{s.session.Copy()}
}

func (s *Session) GetCollection(col string) *mgo.Collection {
	return s.session.DB("capstone").C(col)
}

func (s *Session) Close() {
	if s.session != nil {
		s.session.Close()
	}
}

func (s *Session) DropDatabase(db string) error {
	if s.session != nil {
		return s.session.DB(db).DropDatabase()
	}
	return nil
}
