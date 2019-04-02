package mongo

import (
	mgo "gopkg.in/mgo.v2"
)

type Session struct {
	session *mgo.Session
}

const (
	MongoDBHosts = "mongodb:27017"
	database     = "mongodb"
	userName     = "root"
	password     = "pwd"
	testDatabase = "testdb"
)
const mongoURI = "mongodb://root:pwd@mongodb:27017/"

func NewSession(url string) (*Session, error) {

	/* mongoDBDialInfo := &mgo.DialInfo{
		Addrs:    []string{MongoDBHosts},
		Timeout:  60 * time.Second,
		Database: database,
		Username: userName,
		Password: password,
	} */

	session, err := mgo.Dial("mongodb:27017")
	//	session, err := mgo.Dial(mongoURI)
	//	session, err := mgo.DialWithInfo(mongoDBDialInfo)
	if err != nil {
		return nil, err
	}
	//	session.Login(&mgo.Credential{Username: userName, Password: password})
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
