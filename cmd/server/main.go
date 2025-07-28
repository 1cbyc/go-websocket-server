package main

import (
	"net/http"
	"os"
	"time"

	"github.com/1cbyc/go-websocket-server/internal/auth"
	"github.com/1cbyc/go-websocket-server/internal/config"
	"github.com/1cbyc/go-websocket-server/internal/handler"
	"github.com/1cbyc/go-websocket-server/internal/logger"
	"github.com/1cbyc/go-websocket-server/internal/server"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

func main() {
	cfg := config.Load()
	log := logger.New(cfg.LogLevel)
	s := server.New(cfg, log)
	jwtSecret := os.Getenv("WS_JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "changeme"
	}
	expiry := 24 * time.Hour
	a := auth.New(jwtSecret, expiry)
	r := mux.NewRouter()
	r.Handle("/ws", handler.WebSocketHandler(s, a))
	r.Handle("/history", handler.HistoryHandler(s, a))
	r.Handle("/presence/online", handler.PresenceOnlineHandler(s, a))
	r.Handle("/presence/{userID}", handler.PresenceUserHandler(s, a))
	r.Handle("/rooms", handler.RoomsHandler(s, a))
	r.Handle("/rooms/{roomID}", handler.RoomHandler(s, a))
	r.Handle("/rooms/{roomID}/join", handler.RoomJoinHandler(s, a))
	r.Handle("/rooms/{roomID}/leave", handler.RoomLeaveHandler(s, a))
	r.Handle("/rooms/{roomID}/history", handler.RoomHistoryHandler(s, a))
	log.Info("server starting", zap.String("addr", cfg.Addr))
	http.ListenAndServe(cfg.Addr, r)
}
