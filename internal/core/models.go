package core

import (
	"crypto/rand"
	"encoding/hex"
	"sort"
	"time"
)

type Application struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Repository  string            `json:"repository"`
	Branch      string            `json:"branch"`
	Dockerfile  string            `json:"dockerfile"`
	Context     string            `json:"context"`
	Port        int               `json:"port"`
	PublishPort int               `json:"publishPort"`
	HealthPath  string            `json:"healthPath"`
	Env         map[string]string `json:"env,omitempty"`
	EnvKeys     []string          `json:"envKeys,omitempty"`
	Status      string            `json:"status"`
	CreatedAt   time.Time         `json:"createdAt"`
	UpdatedAt   time.Time         `json:"updatedAt"`
}

func (a Application) Public() Application {
	a.EnvKeys = make([]string, 0, len(a.Env))
	for key := range a.Env {
		a.EnvKeys = append(a.EnvKeys, key)
	}
	sort.Strings(a.EnvKeys)
	a.Env = nil
	return a
}

type Deployment struct {
	ID         string     `json:"id"`
	AppID      string     `json:"appId"`
	Commit     string     `json:"commit,omitempty"`
	Image      string     `json:"image,omitempty"`
	Status     string     `json:"status"`
	Trigger    string     `json:"trigger"`
	Logs       string     `json:"logs,omitempty"`
	Error      string     `json:"error,omitempty"`
	StartedAt  time.Time  `json:"startedAt"`
	FinishedAt *time.Time `json:"finishedAt,omitempty"`
}

func NewID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return time.Now().UTC().Format("20060102150405.000000000")
	}
	return hex.EncodeToString(b[:])
}
