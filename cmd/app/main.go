package main

import (
	"log"

	"github.com/Trirandom/capstone/server/pkg/crypto"
	"github.com/Trirandom/capstone/server/pkg/mongo"
	"github.com/Trirandom/capstone/server/pkg/server"
)

func main() {
	ms, err := mongo.NewSession("127.0.0.1:27017")
	if err != nil {
		log.Fatalln("unable to connect to mongodb")
	}
	defer ms.Close()

	h := crypto.Hash{}
	u := mongo.NewUserService(ms.Copy(), "capsotone_web_server", "user", &h)
	s := server.NewServer(u)

	s.Start()
}
