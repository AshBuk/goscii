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
	LevelID     string    `json:"level_id"`
	Difficulty  string    `json:"difficulty"`
	CompletedAt time.Time `json:"completed_at"`
}

// TopicStat tracks completed missions for one Go topic.
type TopicStat struct {
	Completed []CompletedRun `json:"completed"`
}

// Progress is the player's persistent state across sessions.
type Progress struct {
	AdventureCheckpoint string               `json:"adventure_checkpoint,omitempty"`
	Topics              map[string]TopicStat `json:"topics,omitempty"`
}

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

// RecordCompletion adds a completed run to the topic stats and saves progress.
func (p *Progress) RecordCompletion(topicSlug, levelID, difficulty string) {
	stat := p.Topics[topicSlug]
	stat.Completed = append(stat.Completed, CompletedRun{
		LevelID:     levelID,
		Difficulty:  difficulty,
		CompletedAt: time.Now(),
	})
	p.Topics[topicSlug] = stat
}
