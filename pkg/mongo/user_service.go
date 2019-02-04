package mongo

import (
	root "github.com/Trirandom/capstone/server/pkg"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type UserService struct {
	collection *mgo.Collection
	hash       root.Hash
}

func NewUserService(session *Session, dbName string, collectionName string, hash root.Hash) *UserService {
	collection := session.GetCollection(dbName, collectionName)
	collection.EnsureIndex(userModelIndex())
	return &UserService{collection, hash}
}

func (p *UserService) CreateUser(u *root.User) error {
	user := newUserModel(u)
	return p.collection.Insert(&user)
}

func (p *UserService) GetByUsername(username string) (*root.User, error) {
	model := userModel{}
	err := p.collection.Find(bson.M{"username": username}).One(&model)
	return model.toRootUser(), err
}
