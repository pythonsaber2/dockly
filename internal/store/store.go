package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/pythonsaber2/dockly/internal/core"
)

type state struct {
	Applications map[string]core.Application `json:"applications"`
	Deployments  map[string]core.Deployment  `json:"deployments"`
	AdminHash    string                      `json:"adminHash,omitempty"`
	WebhookToken string                      `json:"webhookToken,omitempty"`
}

type Store struct {
	mu   sync.RWMutex
	path string
	data state
}

func Open(path string) (*Store, error) {
	s := &Store{path: path, data: state{Applications: map[string]core.Application{}, Deployments: map[string]core.Deployment{}}}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, &s.data); err != nil {
		return nil, fmt.Errorf("decode store: %w", err)
	}
	if s.data.Applications == nil {
		s.data.Applications = map[string]core.Application{}
	}
	if s.data.Deployments == nil {
		s.data.Deployments = map[string]core.Deployment{}
	}
	return s, nil
}

func (s *Store) persistLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func (s *Store) CreateApplication(a core.Application) (core.Application, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if a.Name == "" || a.Repository == "" {
		return a, errors.New("name and repository are required")
	}
	for _, existing := range s.data.Applications {
		if existing.Name == a.Name {
			return a, errors.New("application name already exists")
		}
	}
	now := time.Now().UTC()
	a.ID = core.NewID()
	a.CreatedAt = now
	a.UpdatedAt = now
	if a.Branch == "" {
		a.Branch = "main"
	}
	if a.Dockerfile == "" {
		a.Dockerfile = "Dockerfile"
	}
	if a.Context == "" {
		a.Context = "."
	}
	if a.HealthPath == "" {
		a.HealthPath = "/"
	}
	if a.Status == "" {
		a.Status = "idle"
	}
	if a.Env == nil {
		a.Env = map[string]string{}
	}
	s.data.Applications[a.ID] = a
	return a, s.persistLocked()
}

func (s *Store) Application(id string) (core.Application, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	a, ok := s.data.Applications[id]
	return a, ok
}
func (s *Store) Applications() []core.Application {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]core.Application, 0, len(s.data.Applications))
	for _, a := range s.data.Applications {
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out
}
func (s *Store) UpdateApplication(a core.Application) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data.Applications[a.ID]; !ok {
		return os.ErrNotExist
	}
	a.UpdatedAt = time.Now().UTC()
	s.data.Applications[a.ID] = a
	return s.persistLocked()
}
func (s *Store) DeleteApplication(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data.Applications[id]; !ok {
		return os.ErrNotExist
	}
	delete(s.data.Applications, id)
	for did, d := range s.data.Deployments {
		if d.AppID == id {
			delete(s.data.Deployments, did)
		}
	}
	return s.persistLocked()
}

func (s *Store) SaveDeployment(d core.Deployment) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Deployments[d.ID] = d
	return s.persistLocked()
}
func (s *Store) Deployment(id string) (core.Deployment, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, ok := s.data.Deployments[id]
	return d, ok
}
func (s *Store) Deployments(appID string) []core.Deployment {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []core.Deployment{}
	for _, d := range s.data.Deployments {
		if appID == "" || d.AppID == appID {
			out = append(out, d)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].StartedAt.After(out[j].StartedAt) })
	return out
}
func (s *Store) AdminHash() string { s.mu.RLock(); defer s.mu.RUnlock(); return s.data.AdminHash }
func (s *Store) SetAdminHash(hash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.AdminHash = hash
	return s.persistLocked()
}
func (s *Store) WebhookToken() string { s.mu.RLock(); defer s.mu.RUnlock(); return s.data.WebhookToken }
func (s *Store) SetWebhookToken(token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.WebhookToken = token
	return s.persistLocked()
}
