package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

type Payload struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Category    string   `json:"category"`
	Tags        []string `json:"tags"`
	Script      string   `json:"script"`
	Description string   `json:"description,omitempty"`
	Author      string   `json:"author,omitempty"`
	Version     string   `json:"version,omitempty"`
	CreatedAt   string   `json:"created_at,omitempty"`
	UpdatedAt   string   `json:"updated_at,omitempty"`
}

var (
	payloadDir string
	payloadMu  sync.RWMutex
	payloadCache map[string]*Payload
)

func InitPayloads(dir string) error {
	payloadDir = dir
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return reloadPayloadCache()
}

func reloadPayloadCache() error {
	payloadMu.Lock()
	defer payloadMu.Unlock()

	payloadCache = make(map[string]*Payload)
	entries, _ := os.ReadDir(payloadDir)
	for _, e := range entries {
		if filepath.Ext(e.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(payloadDir, e.Name()))
		if err != nil {
			continue
		}
		var p Payload
		if err := json.Unmarshal(data, &p); err != nil {
			continue
		}
		if p.Tags == nil {
			p.Tags = []string{}
		}
		payloadCache[p.ID] = &p
	}
	return nil
}

func GetAllPayloads() []Payload {
	payloadMu.RLock()
	defer payloadMu.RUnlock()

	list := make([]Payload, 0, len(payloadCache))
	for _, p := range payloadCache {
		list = append(list, *p)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].UpdatedAt > list[j].UpdatedAt
	})
	return list
}

func GetPayloadByID(id string) (*Payload, bool) {
	payloadMu.RLock()
	defer payloadMu.RUnlock()
	p, ok := payloadCache[id]
	if !ok {
		return nil, false
	}
	cp := *p
	return &cp, true
}

func SavePayload(p *Payload) error {
	now := time.Now().Format(time.RFC3339)
	if p.ID == "" {
		p.ID = fmt.Sprintf("%d", time.Now().UnixMilli())
	}
	if p.Tags == nil {
		p.Tags = []string{}
	}

	payloadMu.Lock()
	if _, exists := payloadCache[p.ID]; !exists {
		p.CreatedAt = now
	}
	payloadMu.Unlock()
	p.UpdatedAt = now

	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(payloadDir, p.ID+".json")
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}

	payloadMu.Lock()
	cp := *p
	payloadCache[p.ID] = &cp
	payloadMu.Unlock()
	return nil
}

func DeletePayload(id string) error {
	path := filepath.Join(payloadDir, id+".json")
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	payloadMu.Lock()
	delete(payloadCache, id)
	payloadMu.Unlock()
	return nil
}

func GetPayloadCategories(extra []string) []string {
	payloadMu.RLock()
	seen := make(map[string]bool)
	for _, p := range payloadCache {
		if p.Category != "" {
			seen[p.Category] = true
		}
	}
	payloadMu.RUnlock()
	for _, c := range extra {
		seen[c] = true
	}
	cats := make([]string, 0, len(seen))
	for c := range seen {
		cats = append(cats, c)
	}
	sort.Strings(cats)
	return cats
}

func GetPayloadTags(extra []string) []string {
	payloadMu.RLock()
	seen := make(map[string]bool)
	for _, p := range payloadCache {
		for _, t := range p.Tags {
			seen[t] = true
		}
	}
	payloadMu.RUnlock()
	for _, t := range extra {
		seen[t] = true
	}
	tags := make([]string, 0, len(seen))
	for t := range seen {
		tags = append(tags, t)
	}
	sort.Strings(tags)
	return tags
}

// MigrateOld migrates from old payloads.json flat file format
func MigrateOld(oldFile string) error {
	data, err := os.ReadFile(oldFile)
	if err != nil {
		return nil
	}
	var old []Payload
	if err := json.Unmarshal(data, &old); err != nil {
		return nil
	}
	count := 0
	for _, p := range old {
		cp := p
		if _, exists := GetPayloadByID(cp.ID); !exists {
			if err := SavePayload(&cp); err == nil {
				count++
			}
		}
	}
	if count > 0 {
		os.Rename(oldFile, oldFile+".migrated")
	}
	return nil
}
