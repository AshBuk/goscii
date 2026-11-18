// Copyright 2026 Asher Buk
// SPDX-License-Identifier: Apache-2.0
// https://github.com/AshBuk/goscii

package engine

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// CompletedRun records a single completed AI-generated mission.
type CompletedRun struct {
	MissionID   string    `json:"mission_id"`
	Difficulty  string    `json:"difficulty"`
	Beat        string    `json:"beat,omitempty"`
	Kind        string    `json:"kind,omitempty"`
	Runs        int       `json:"runs,omitempty"` // runs it took, the passing one included
	CompletedAt time.Time `json:"completed_at"`
}

// TopicStat tracks completed missions for one Go topic.
type TopicStat struct {
	Completed []CompletedRun `json:"completed"`
}

// Progress is the player's persistent state across sessions.
type Progress struct {
	AdventureCheckpoint string               `json:"adventure_checkpoint,omitempty"`
	AdventureDone       int                  `json:"adventure_done,omitempty"`
	Topics              map[string]TopicStat `json:"topics,omitempty"`
	Station             Station              `json:"station,omitempty"`
	Logs                []LogEntry           `json:"logs,omitempty"` // crew log fragments recovered, in order
}

// LogEntry is one recovered crew log fragment, as GOSCII rendered it.
type LogEntry struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

// DataDir returns the directory holding config, progress and the mission archive.
func DataDir() (string, error) { return dataDir() }

func dataDir() (string, error) {
	if d := os.Getenv("XDG_DATA_HOME"); d != "" {
		return filepath.Join(d, "goscii"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("locate home directory: %w", err)
	}
	return filepath.Join(home, ".local", "share", "goscii"), nil
}

func progressPath() (string, error) {
	dir, err := dataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "progress.json"), nil
}

func LoadProgress() (*Progress, error) {
	path, err := progressPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Progress{
				Topics: make(map[string]TopicStat),
			}, nil
		}
		return nil, err
	}
	var p Progress
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	if p.Topics == nil {
		p.Topics = make(map[string]TopicStat)
	}
	return &p, nil
}

func SaveProgress(p *Progress) error {
	path, err := progressPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// RecordCompletion adds a completed run to the topic stats.
func (p *Progress) RecordCompletion(topicSlug string, run CompletedRun) {
	if p.Topics == nil {
		p.Topics = make(map[string]TopicStat)
	}
	if run.CompletedAt.IsZero() {
		run.CompletedAt = time.Now()
	}
	stat := p.Topics[topicSlug]
	stat.Completed = append(stat.Completed, run)
	p.Topics[topicSlug] = stat
}

// BeatRuns returns how often a beat was completed and the most runs any completion took.
func (p *Progress) BeatRuns(topicSlug, beat string) (done, worst int) {
	if p == nil {
		return 0, 0
	}
	for _, r := range p.Topics[topicSlug].Completed {
		if r.Beat == beat {
			done++
			worst = max(worst, r.Runs)
		}
	}
	return done, worst
}

// Repaired counts every mission the astronaut has completed.
func (p *Progress) Repaired() int {
	if p == nil {
		return 0
	}
	n := p.AdventureDone
	for _, s := range p.Topics {
		n += len(s.Completed)
	}
	return n
}
