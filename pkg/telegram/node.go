package telegram

import (
	"encoding/json"
	"fmt"

	"github.com/docker/libkv/store"
)

// NodeExported saves the exported node
type NodeExported struct {
	ID    string `json:"node_id"`
	Name  string `json:"name"`
	Owner string `json:"owner_id"`
}

// NodeStore writes the users to a libkv store backend
type NodeStore struct {
	kv store.Store
}

// NewNodeStore stores telegram chats in the provided kv backend
func NewNodeStore(kv store.Store) (*NodeStore, error) {
	return &NodeStore{kv: kv}, nil
}

const telegramNodesDirectory = "telegram/nodes"

// List all nodes saved in the kv backend
func (s *NodeStore) List() ([]NodeExported, error) {
	kvPairs, err := s.kv.List(telegramNodesDirectory)
	if err != nil {
		return nil, err
	}

	var nodes []NodeExported
	for _, kv := range kvPairs {
		var n NodeExported
		if err := json.Unmarshal(kv.Value, &n); err != nil {
			return nil, err
		}
		nodes = append(nodes, n)
	}

	return nodes, nil
}

// Add a telegram node to the kv backend
func (s *NodeStore) Add(n NodeExported) error {
	b, err := json.Marshal(n)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("%s/%s", telegramNodesDirectory, n.ID)

	return s.kv.Put(key, b, nil)
}

// Remove a telegram nodes from the kv backend
func (s *NodeStore) Remove(n NodeExported) error {
	key := fmt.Sprintf("%s/%s", telegramNodesDirectory, n.ID)
	return s.kv.Delete(key)
}
