package main

import (
	"errors"
	"fmt"
	irc "github.com/fluffle/goirc/client"
	"github.com/nu7hatch/gouuid"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	//"github.com/gophergala/cheppirc/theme"
	"github.com/mimrock/cheppirc/session"
	//"github.com/gophergala/cheppirc/target"
	"github.com/gorilla/websocket"
)

type SessionList struct {
	Sessions map[string]session.Session
}

type chatHandler struct {
	sessionList *SessionList
}

type loginHandler struct {
}

type connectHandler struct {
	sessionList *SessionList
}

type wsHandler struct {
	sessionList *SessionList
	WsClose     chan bool
}

type sendHandler struct {
	sessionList *SessionList
	WsClose     chan bool
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func (c *chatHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	accessMsg := fmt.Sprintf("%v %v from %v Headers: %+v", r.Method, r.RequestURI, r.RemoteAddr, r.Header)
	log.Println(accessMsg)

	r.ParseForm()
	session := getSession(r.Form, c.sessionList)
	//TODO: validate if uuid exists
	if session == nil {
		http.Redirect(w, r, "login", 302)
		return
	}

	t := template.Must(template.ParseFiles("templates/chat.html"))
	session.RLock()
	t.Execute(w, session)
	session.RUnlock()

	//w.Write([]byte("Hello IRC"))
	//w.Write(data)
}

func (c *loginHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	accessMsg := fmt.Sprintf("%v %v from %v Headers: %+v", r.Method, r.RequestURI, r.RemoteAddr, r.Header)
	log.Println(accessMsg)

	t := template.Must(template.ParseFiles("templates/login.html"))
	t.Execute(w, "CHEPPIRC")
}

func (c *connectHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	log.Println(r.Form)
	nick := r.Form.Get("nick")
	channel := r.Form.Get("channel")
	server := r.Form.Get("server")
	port := r.Form.Get("port")

	session, err := newSession(nick, channel, server, port)
	if err != nil {
		w.Write([]byte("{\"success\": false, \"message\": \"" + err.Error() + "\"}"))
	}

	c.sessionList.Sessions[session.Uuid] = *session

	w.Write([]byte("{\"success\": true, \"message\": \"" + session.Uuid + "\"}"))
}

func (wh *wsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println("\n *** Handle WS *** \n")
	r.ParseForm()

	session := getSession(r.Form, wh.sessionList)
	if session == nil {
		log.Println("NO SESSION IN WS CONNECTION")
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("ERROR IN CONNECTION:", err.Error())
		return
	}
	for {
		log.Println("Waiting...")
		select {
		case chatMessage := <-session.Updater:
			//escapedMessage := template.HTMLEscapeString(string(chatMessage))
			//err := conn.WriteMessage(websocket.TextMessage, []byte(escapedMessage))
			log.Println("\nWRITEMESSAGE:", string(chatMessage))
			err := conn.WriteMessage(websocket.TextMessage, chatMessage)
			if err != nil {
				log.Println("ERROR WRITING TO SOCKET:", err.Error())
				return
			} else {
				log.Println("\n *** SUCCESS *** \n")
			}
		case <-wh.WsClose:
			return
		}

	}
}

func (s *sendHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println("\n **** sendHandler ServeHTTP *** \n")
	r.ParseForm()

	session := getSession(r.Form, s.sessionList)
	if session == nil {
		log.Println("NO SESSION IN WS CONNECTION")
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("ERROR IN CONNECTION:", err.Error())
		return
	}

	for {
		log.Println("\n--READ--\n")
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			log.Println("SENDHANDLER ERROR READING SOCKET:", err.Error())
			s.WsClose <- true
			return
		}
		log.Println("DEBUG READ:", messageType, " -- ", string(p))
		data := strings.Split(string(p), "||")
		session.C.Privmsg(data[0], data[1])
		session.AddMessage(data[0], session.Nick, data[1], "self", session.Updater)
	}
}

func newChatHandler(s *SessionList) *chatHandler {
	c := &chatHandler{s}
	return c
}

func newLoginHandler() *loginHandler {
	c := &loginHandler{}
	return c
}

func newConnectHandler(s *SessionList) *connectHandler {
	c := &connectHandler{s}
	return c
}

func newWsHandler(s *SessionList, wsClose chan bool) *wsHandler {
	w := &wsHandler{s, wsClose}
	return w
}

func newSendHandler(s *SessionList, wsClose chan bool) *sendHandler {
	w := &sendHandler{s, wsClose}
	return w
}

func newSession(nick, channel, server, port string) (*session.Session, error) {
	cfg := irc.NewConfig(nick)
	cfg.SSL = false
	cfg.Server = server + ":" + port
	cfg.NewNick = func(n string) string { return n + "^" }
	c := irc.Client(cfg)

	log.Println(c.String())
	id, _ := uuid.NewV4()
	session := &session.Session{}
	session.Uuid = id.String()
	session.C = c
	session.Updater = make(chan []byte, 25)
	log.Println("\nUUID:", id.String())
	log.Println("\nCFG:", cfg)
	session.AddMessage(channel, "", "Connecting to "+channel, "status", session.Updater)

	c.HandleFunc("connected",
		func(conn *irc.Conn, line *irc.Line) {
			log.Println("Connected to", line.Raw)
			conn.Join(channel)
			session.AddMessage(channel, "", "Now talking on "+channel, "status", session.Updater)
			conn.Who(channel)
		})

	c.HandleFunc("privmsg",
		func(conn *irc.Conn, line *irc.Line) {
			log.Println("PRIVMSG - Raw:", line.Raw, "Nick:", line.Nick, "Src:", line.Src, "Args:", line.Args, "time:", line.Time)
			session.AddMessage(line.Args[0], line.Nick, line.Args[1], "user", session.Updater)
		})

	c.HandleFunc("352",
		func(conn *irc.Conn, line *irc.Line) {
			log.Println("352 - RAW:", line.Raw)
			session.SetUsers(line.Args[1], line.Args[5], line.Args[3]+" "+line.Args[4])
		})

	c.HandleFunc("315",
		func(conn *irc.Conn, line *irc.Line) {
			log.Println("315 - RAW:", line.Raw)
			session.AddMessage(line.Args[1], "", "reload", "hidden", session.Updater)
		})

	if err := c.Connect(); err != nil {
		return nil, errors.New("Connection error: " + err.Error())
	}

	return session, nil
}

func getSession(values url.Values, sessionList *SessionList) *session.Session {
	uuid := values.Get("session")
	if len(uuid) < 1 {
		return nil
	}

	if s, ok := sessionList.Sessions[uuid]; ok {
		return &s
	}

	return nil
}

func main() {
	log.Println("Starting up server...")
	sessionList := new(SessionList)
	sessionList.Sessions = make(map[string]session.Session)

	closeWs := make(chan bool)

	mux := http.NewServeMux()
	mux.Handle("/sendws", newSendHandler(sessionList, closeWs))
	mux.Handle("/ws", newWsHandler(sessionList, closeWs))
	mux.Handle("/chat", newChatHandler(sessionList))
	mux.Handle("/login", newLoginHandler())
	mux.Handle("/connect", newConnectHandler(sessionList))
	
	// Make the css and js files available in subdirectories
	mux.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir("css"))))	
	mux.Handle("/js/", http.StripPrefix("/js/", http.FileServer(http.Dir("js"))))	
	
	log.Println("Listening...")

	http.ListenAndServe(":"+strconv.Itoa(8081), mux)
}
