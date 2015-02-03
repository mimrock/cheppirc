package target

import (
	"github.com/gophergala/cheppirc/message"
	"strings"
)

type Target struct {
	Name     string
	Type     string
	Messages []message.Message
}

func (t *Target) AddMessage(m message.Message) {
	t.Messages = append(t.Messages, m)
}

func NewTarget(name string) *Target {
	var t Target
	if name[0] == 35 {
		//If the target start with a # then it's a channel
		targetName := strings.Trim(name, "# ")
		t = Target{targetName, "channel", []message.Message{}}
	} else {
		t = Target{name, "other", []message.Message{}}
	}
	//t.Messages = []Message{}
	return &t
}
