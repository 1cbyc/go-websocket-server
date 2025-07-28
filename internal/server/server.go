package server

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/1cbyc/go-websocket-server/internal/config"
	"github.com/1cbyc/go-websocket-server/internal/model"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/net/websocket"
)

type Server struct {
	conns     map[*websocket.Conn]bool
	connRooms map[*websocket.Conn]string
	mu        sync.Mutex
	cfg       *config.Config
	log       *zap.Logger
	store     model.MessageStore
	presence  model.PresenceStore
	rooms     model.RoomStore
}

func New(cfg *config.Config, log *zap.Logger) *Server {
	store, err := model.NewSQLiteMessageStore(cfg.DBDSN)
	if err != nil {
		log.Fatal("failed to init message store", zap.Error(err))
	}
	presence, err := model.NewSQLitePresenceStore(cfg.DBDSN)
	if err != nil {
		log.Fatal("failed to init presence store", zap.Error(err))
	}
	rooms, err := model.NewSQLiteRoomStore(cfg.DBDSN)
	if err != nil {
		log.Fatal("failed to init room store", zap.Error(err))
	}
	return &Server{
		conns:     make(map[*websocket.Conn]bool),
		connRooms: make(map[*websocket.Conn]string),
		cfg:       cfg,
		log:       log,
		store:     store,
		presence:  presence,
		rooms:     rooms,
	}
}

func (s *Server) HandleWS(ws *websocket.Conn) {
	s.mu.Lock()
	s.conns[ws] = true
	s.mu.Unlock()
	userID := ws.Request().Header.Get("X-User-ID")
	s.presence.Set(context.Background(), &model.Presence{
		UserID:   userID,
		Online:   true,
		LastSeen: time.Now().Unix(),
	})
	s.readLoop(ws)
}

func (s *Server) readLoop(ws *websocket.Conn) {
	dec := json.NewDecoder(ws)
	userID := ws.Request().Header.Get("X-User-ID")
	for {
		var msg model.Message
		err := dec.Decode(&msg)
		if err != nil {
			if err == io.EOF {
				s.removeConn(ws)
				s.presence.Set(context.Background(), &model.Presence{
					UserID:   userID,
					Online:   false,
					LastSeen: time.Now().Unix(),
				})
				return
			}
			s.log.Error("read error", zap.Error(err))
			continue
		}
		if msg.Content == "" || msg.UserID == "" || msg.RoomID == "" {
			s.log.Warn("invalid message", zap.Any("msg", msg))
			continue
		}
		msg.ID = uuid.NewString()
		msg.Timestamp = time.Now().Unix()
		s.store.Save(context.Background(), &msg)
		s.mu.Lock()
		s.connRooms[ws] = msg.RoomID
		s.mu.Unlock()
		s.broadcast(msg)
	}
}

func (s *Server) broadcast(msg model.Message) {
	b, err := json.Marshal(msg)
	if err != nil {
		s.log.Error("marshal error", zap.Error(err))
		return
	}
	s.mu.Lock()
	for ws := range s.conns {
		roomID := s.connRooms[ws]
		if roomID == msg.RoomID {
			go func(ws *websocket.Conn) {
				if _, err := ws.Write(b); err != nil {
					s.log.Error("write error", zap.Error(err))
				}
			}(ws)
		}
	}
	s.mu.Unlock()
}

func (s *Server) removeConn(ws *websocket.Conn) {
	s.mu.Lock()
	delete(s.conns, ws)
	delete(s.connRooms, ws)
	s.mu.Unlock()
}

func (s *Server) Start() {
	http.Handle("/ws", websocket.Handler(s.HandleWS))
	s.log.Info("server starting", zap.String("addr", s.cfg.Addr))
	http.ListenAndServe(s.cfg.Addr, nil)
}

func (s *Server) Store() model.MessageStore {
	return s.store
}

func (s *Server) Presence() model.PresenceStore {
	return s.presence
}

func (s *Server) RoomStore() model.RoomStore {
	return s.rooms
}
