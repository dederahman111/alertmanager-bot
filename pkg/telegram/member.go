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
	Username string       `json:"username"`
	Level    HandleLevel  `json:"level"`
	Chat     telebot.Chat `json:"chat"`
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
func (s *MemberStore) List() ([]Member, error) {
	kvPairs, err := s.kv.List(telegramMembersDirectory)
	if err != nil {
		return nil, err
	}

	var members []Member
	for _, kv := range kvPairs {
		var m Member
		if err := json.Unmarshal(kv.Value, &m); err != nil {
			return nil, err
		}
		members = append(members, m)
	}

	return members, nil
}

// Add a telegram member to the kv backend
func (s *MemberStore) Add(m Member) error {
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("%s/%s", telegramMembersDirectory, m.Username)

	return s.kv.Put(key, b, nil)
}

// Remove a telegram members from the kv backend
func (s *MemberStore) Remove(m Member) error {
	key := fmt.Sprintf("%s/%s", telegramMembersDirectory, m.Username)
	return s.kv.Delete(key)
}

// GetMembersByChat helps getting members by chat ID
func (s *MemberStore) GetMembersByChat(chat telebot.Chat) ([]Member, error) {
	var ret []Member
	members, err := s.List()
	if err != nil {
		return ret, err
	}

	for _, ms := range members {
		if ms.Chat.ID == chat.ID {
			ret = append(ret, ms)
		}
	}
	return ret, err
}

// GetRandomMemberByChatandLevel get random member by level
func (s *MemberStore) GetRandomMemberByChatandLevel(chat telebot.Chat, level string) (Member, error) {
	var ret Member
	var groupByLevel []Member
	groupByChat, err := s.GetMembersByChat(chat)
	if err != nil {
		return ret, err
	}
	for _, mr := range groupByChat {
		if mr.Level == HandleLevel(level) {
			groupByLevel = append(groupByLevel, mr)
		}
	}
	rand.Seed(time.Now().UnixNano())
	choosen := groupByLevel[rand.Intn(len(groupByLevel))]
	return choosen, nil
}
