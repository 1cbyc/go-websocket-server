package model

import (
	"context"
	"database/sql"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

type User struct {
	ID   string
	Name string
}

type Message struct {
	ID        string
	UserID    string
	RoomID    string
	Content   string
	Timestamp int64
}

type Room struct {
	ID      string
	Name    string
	Members []string
}

type Presence struct {
	UserID   string
	Online   bool
	LastSeen int64
}

type MessageStore interface {
	Save(ctx context.Context, msg *Message) error
	List(ctx context.Context, limit int) ([]*Message, error)
	ListByRoom(ctx context.Context, roomID string, limit int) ([]*Message, error)
}

type PresenceStore interface {
	Set(ctx context.Context, p *Presence) error
	Get(ctx context.Context, userID string) (*Presence, error)
	ListOnline(ctx context.Context) ([]*Presence, error)
}

type RoomStore interface {
	Create(ctx context.Context, room *Room) error
	Get(ctx context.Context, id string) (*Room, error)
	List(ctx context.Context) ([]*Room, error)
	AddMember(ctx context.Context, roomID, userID string) error
	RemoveMember(ctx context.Context, roomID, userID string) error
}

type SQLiteMessageStore struct {
	db *sql.DB
}

type SQLitePresenceStore struct {
	db *sql.DB
}

type SQLiteRoomStore struct {
	db *sql.DB
}

func NewSQLiteMessageStore(dsn string) (*SQLiteMessageStore, error) {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS messages (id TEXT PRIMARY KEY, user_id TEXT, room_id TEXT, content TEXT, timestamp INTEGER)`)
	if err != nil {
		return nil, err
	}
	return &SQLiteMessageStore{db: db}, nil
}

func (s *SQLiteMessageStore) Save(ctx context.Context, msg *Message) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO messages (id, user_id, room_id, content, timestamp) VALUES (?, ?, ?, ?, ?)`, msg.ID, msg.UserID, msg.RoomID, msg.Content, msg.Timestamp)
	return err
}

func (s *SQLiteMessageStore) List(ctx context.Context, limit int) ([]*Message, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, user_id, room_id, content, timestamp FROM messages ORDER BY timestamp DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var msgs []*Message
	for rows.Next() {
		var m Message
		err := rows.Scan(&m.ID, &m.UserID, &m.RoomID, &m.Content, &m.Timestamp)
		if err != nil {
			return nil, err
		}
		msgs = append(msgs, &m)
	}
	return msgs, nil
}

func (s *SQLiteMessageStore) ListByRoom(ctx context.Context, roomID string, limit int) ([]*Message, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, user_id, room_id, content, timestamp FROM messages WHERE room_id = ? ORDER BY timestamp DESC LIMIT ?`, roomID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var msgs []*Message
	for rows.Next() {
		var m Message
		err := rows.Scan(&m.ID, &m.UserID, &m.RoomID, &m.Content, &m.Timestamp)
		if err != nil {
			return nil, err
		}
		msgs = append(msgs, &m)
	}
	return msgs, nil
}

func NewSQLitePresenceStore(dsn string) (*SQLitePresenceStore, error) {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS presence (user_id TEXT PRIMARY KEY, online INTEGER, last_seen INTEGER)`)
	if err != nil {
		return nil, err
	}
	return &SQLitePresenceStore{db: db}, nil
}

func (s *SQLitePresenceStore) Set(ctx context.Context, p *Presence) error {
	_, err := s.db.ExecContext(ctx, `INSERT OR REPLACE INTO presence (user_id, online, last_seen) VALUES (?, ?, ?)`, p.UserID, boolToInt(p.Online), p.LastSeen)
	return err
}

func (s *SQLitePresenceStore) Get(ctx context.Context, userID string) (*Presence, error) {
	row := s.db.QueryRowContext(ctx, `SELECT user_id, online, last_seen FROM presence WHERE user_id = ?`, userID)
	var p Presence
	var online int
	err := row.Scan(&p.UserID, &online, &p.LastSeen)
	if err != nil {
		return nil, err
	}
	p.Online = online == 1
	return &p, nil
}

func (s *SQLitePresenceStore) ListOnline(ctx context.Context) ([]*Presence, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT user_id, online, last_seen FROM presence WHERE online = 1`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ps []*Presence
	for rows.Next() {
		var p Presence
		var online int
		err := rows.Scan(&p.UserID, &online, &p.LastSeen)
		if err != nil {
			return nil, err
		}
		p.Online = online == 1
		ps = append(ps, &p)
	}
	return ps, nil
}

func NewSQLiteRoomStore(dsn string) (*SQLiteRoomStore, error) {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS rooms (id TEXT PRIMARY KEY, name TEXT, members TEXT)`)
	if err != nil {
		return nil, err
	}
	return &SQLiteRoomStore{db: db}, nil
}

func (s *SQLiteRoomStore) Create(ctx context.Context, room *Room) error {
	members := strings.Join(room.Members, ",")
	_, err := s.db.ExecContext(ctx, `INSERT INTO rooms (id, name, members) VALUES (?, ?, ?)`, room.ID, room.Name, members)
	return err
}

func (s *SQLiteRoomStore) Get(ctx context.Context, id string) (*Room, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, name, members FROM rooms WHERE id = ?`, id)
	var r Room
	var members string
	err := row.Scan(&r.ID, &r.Name, &members)
	if err != nil {
		return nil, err
	}
	r.Members = []string{}
	if members != "" {
		r.Members = strings.Split(members, ",")
	}
	return &r, nil
}

func (s *SQLiteRoomStore) List(ctx context.Context) ([]*Room, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, members FROM rooms`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var rooms []*Room
	for rows.Next() {
		var r Room
		var members string
		err := rows.Scan(&r.ID, &r.Name, &members)
		if err != nil {
			return nil, err
		}
		r.Members = []string{}
		if members != "" {
			r.Members = strings.Split(members, ",")
		}
		rooms = append(rooms, &r)
	}
	return rooms, nil
}

func (s *SQLiteRoomStore) AddMember(ctx context.Context, roomID, userID string) error {
	r, err := s.Get(ctx, roomID)
	if err != nil {
		return err
	}
	for _, m := range r.Members {
		if m == userID {
			return nil
		}
	}
	r.Members = append(r.Members, userID)
	members := strings.Join(r.Members, ",")
	_, err = s.db.ExecContext(ctx, `UPDATE rooms SET members = ? WHERE id = ?`, members, roomID)
	return err
}

func (s *SQLiteRoomStore) RemoveMember(ctx context.Context, roomID, userID string) error {
	r, err := s.Get(ctx, roomID)
	if err != nil {
		return err
	}
	newMembers := []string{}
	for _, m := range r.Members {
		if m != userID {
			newMembers = append(newMembers, m)
		}
	}
	members := strings.Join(newMembers, ",")
	_, err = s.db.ExecContext(ctx, `UPDATE rooms SET members = ? WHERE id = ?`, members, roomID)
	return err
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
