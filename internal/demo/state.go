package demo

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

type Metric struct {
	Label   string
	Value   string
	Unit    string
	Percent int
}

type Command struct {
	Name        string
	Description string
	Tone        string
}

type Activity struct {
	ID      string
	Message string
	Time    time.Time
	Tone    string
}

type StateSnapshot struct {
	Telemetry      []Metric
	Commands       []Command
	Activities     []Activity
	Health         string
	Profile        string
	Requests       int
	Updated        time.Time
	StatusSequence int
}

type State struct {
	mu             sync.RWMutex
	telemetry      []Metric
	commands       []Command
	activities     []Activity
	health         string
	profile        string
	requests       int
	nextActivityID int
	updated        time.Time
	statusSequence int
}

func NewState() *State {
	now := time.Now().UTC()
	return &State{
		telemetry: []Metric{
			{Label: "CPU load", Value: "42", Unit: "%", Percent: 42},
			{Label: "Memory", Value: "68", Unit: "%", Percent: 68},
			{Label: "Requests", Value: "1,284", Unit: "/min", Percent: 74},
		},
		commands: []Command{
			{Name: "deploy api", Description: "Promote the API service to the next ring.", Tone: "lime"},
			{Name: "rotate keys", Description: "Stage a zero-downtime credential rotation.", Tone: "cyan"},
			{Name: "drain edge-03", Description: "Remove one edge node from traffic.", Tone: "orange"},
			{Name: "inspect queue", Description: "Open the current queue pressure report.", Tone: "cyan"},
		},
		activities: []Activity{
			{ID: "activity-1", Message: "Edge health check completed", Time: now.Add(-3 * time.Minute), Tone: "lime"},
			{ID: "activity-2", Message: "Certificate renewal queued", Time: now.Add(-11 * time.Minute), Tone: "cyan"},
			{ID: "activity-3", Message: "Worker pool scaled to 12", Time: now.Add(-19 * time.Minute), Tone: "orange"},
		},
		health:         "Nominal",
		profile:        "production-east",
		nextActivityID: 4,
		updated:        now,
	}
}

func (s *State) Snapshot() StateSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return StateSnapshot{
		Telemetry:      append([]Metric(nil), s.telemetry...),
		Commands:       append([]Command(nil), s.commands...),
		Activities:     append([]Activity(nil), s.activities...),
		Health:         s.health,
		Profile:        s.profile,
		Requests:       s.requests,
		Updated:        s.updated,
		StatusSequence: s.statusSequence,
	}
}

func (s *State) RefreshTelemetry() StateSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requests++
	load := 42 + s.requests%7
	memory := 68 + s.requests%4
	requests := 1284 + s.requests*3
	s.telemetry[0].Value = fmt.Sprintf("%d", load)
	s.telemetry[0].Percent = load
	s.telemetry[1].Value = fmt.Sprintf("%d", memory)
	s.telemetry[1].Percent = memory
	s.telemetry[2].Value = fmt.Sprintf("%s", formatNumber(requests))
	s.telemetry[2].Percent = 74 + s.requests%8
	s.updated = time.Now().UTC()
	return s.snapshotLocked()
}

func (s *State) Search(query string) []Command {
	s.mu.RLock()
	defer s.mu.RUnlock()
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return nil
	}
	results := make([]Command, 0, len(s.commands))
	for _, command := range s.commands {
		if strings.Contains(strings.ToLower(command.Name), query) || strings.Contains(strings.ToLower(command.Description), query) {
			results = append(results, command)
		}
	}
	return results
}

func (s *State) ExecuteCommand(command string) (Command, StateSnapshot, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	command = strings.TrimSpace(command)
	if command == "" {
		return Command{}, s.snapshotLocked(), false
	}
	s.requests++
	s.updated = time.Now().UTC()
	return Command{Name: command, Description: "Accepted by the Signal North control plane.", Tone: "lime"}, s.snapshotLocked(), true
}

func (s *State) AddActivity(message string) (Activity, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	message = strings.TrimSpace(message)
	if message == "" {
		return Activity{}, false
	}
	activity := Activity{
		ID:      fmt.Sprintf("activity-%d", s.nextActivityID),
		Message: message,
		Time:    time.Now().UTC(),
		Tone:    "lime",
	}
	s.nextActivityID++
	s.activities = append([]Activity{activity}, s.activities...)
	s.updated = activity.Time
	return activity, true
}

func (s *State) DeleteActivity(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for index, activity := range s.activities {
		if activity.ID == id {
			s.activities = append(s.activities[:index], s.activities[index+1:]...)
			s.updated = time.Now().UTC()
			return true
		}
	}
	return false
}

func (s *State) UpdateProfile(profile string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	profile = strings.TrimSpace(profile)
	if profile == "" {
		return false
	}
	s.profile = profile
	s.updated = time.Now().UTC()
	return true
}

func (s *State) Status() StateSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.statusSequence++
	s.updated = time.Now().UTC()
	if s.statusSequence%5 == 0 {
		s.health = "Watch"
	} else {
		s.health = "Nominal"
	}
	return s.snapshotLocked()
}

func (s *State) snapshotLocked() StateSnapshot {
	return StateSnapshot{
		Telemetry:      append([]Metric(nil), s.telemetry...),
		Commands:       append([]Command(nil), s.commands...),
		Activities:     append([]Activity(nil), s.activities...),
		Health:         s.health,
		Profile:        s.profile,
		Requests:       s.requests,
		Updated:        s.updated,
		StatusSequence: s.statusSequence,
	}
}

func formatNumber(value int) string {
	if value < 1000 {
		return fmt.Sprintf("%d", value)
	}
	return fmt.Sprintf("%d,%03d", value/1000, value%1000)
}
