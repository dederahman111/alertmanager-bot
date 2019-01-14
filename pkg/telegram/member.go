package telegram

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/docker/libkv/store"
	"github.com/tucnak/telebot"
)

// Member saves the member's telegram info and level in the group
type Member struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Level    string `json:"level"`
}

// Members saves all members of chat with level
type Members struct {
	Chat    telebot.Chat `json:"chat"`
	Members []Member     `json:"members"`
}

// MemberStore writes the users to a libkv store backend
type MemberStore struct {
	kv store.Store
}

// NewMemberStore stores telegram chats in the provided kv backend
func NewMemberStore(kv store.Store) (*MemberStore, error) {
	return &MemberStore{kv: kv}, nil
}

const telegramMembersDirectory = "telegram/members"

// List all members saved in the kv backend
func (s *MemberStore) List() ([]Members, error) {
	kvPairs, err := s.kv.List(telegramMembersDirectory)
	if err != nil {
		return nil, err
	}

	var members []Members
	for _, kv := range kvPairs {
		var m Members
		if err := json.Unmarshal(kv.Value, &m); err != nil {
			return nil, err
		}
		members = append(members, m)
	}

	return members, nil
}

// Add a telegram member to the kv backend
func (s *MemberStore) Add(m Members) error {
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("%s/%d", telegramMembersDirectory, m.Chat.ID)

	return s.kv.Put(key, b, nil)
}

// Remove a telegram members from the kv backend
func (s *MemberStore) Remove(m Members) error {
	key := fmt.Sprintf("%s/%d", telegramMembersDirectory, m.Chat.ID)
	return s.kv.Delete(key)
}

func (s *MemberStore) GetMembersByChat(chat telebot.Chat) (Members, error) {
	var ret Members
	members, err := s.List()
	if err != nil {
		return ret, err
	}

	for _, ms := range members {
		if ms.Chat.ID == chat.ID {
			return ms, nil
		}
	}
	return ret, err
}

// GetRandomMemberByLevel
func (m *Members) GetRandomMemberByLevel(level string) (string, error) {
	var group []Member
	for _, mr := range m.Members {
		if mr.Level == level {
			group = append(group, mr)
		}
	}
	rand.Seed(time.Now().UnixNano())
	choosen := group[rand.Intn(len(group)-1)]
	return choosen.Username, nil
}
