package session

import (
	"encoding/json"
	irc "github.com/fluffle/goirc/client"
	"github.com/gophergala/cheppirc/message"
	"github.com/gophergala/cheppirc/target"
	"github.com/gophergala/cheppirc/user"
	"log"
	"sync"
)

type Session struct {
	Uuid    string
	C       *irc.Conn
	Updater chan []byte
	Targets map[string]target.Target
	Users   map[string]map[string]user.User
	Nick    string
	sync.RWMutex
}

func (s *Session) AddMessage(targ, sender, text string, mtype string, updater chan []byte) {
	log.Println("ADDMESSAGE:", text, "DEBUG USERS:", s.Users)

	s.Lock()
	if _, ok := s.Targets[targ]; !ok {
		log.Println("ADDMESSAGE: Target not found. Target:", targ)
		s.Targets[targ] = *target.NewTarget(targ)
	}

	tempT := s.Targets[targ]
	m := message.Message{sender, text, tempT.Name, mtype}
	tempT.AddMessage(m)
	s.Targets[targ] = tempT
	s.Unlock()

	b, err := json.Marshal(m)
	if err != nil {
		log.Println("Error marshalling message:", err.Error())
	}
	log.Println(s.Targets)
	updater <- b
}

func (s *Session) SetUsers(target, nick, info string) {
	s.Lock()
	if _, ok := s.Users[target]; !ok {
		log.Println("SETUSERS: Target not found. Target:", target)
		s.Users[target] = make(map[string]user.User)
	}
	s.Users[target][nick] = user.User{nick, info}
	s.Unlock()
}
