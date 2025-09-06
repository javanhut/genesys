package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// LocalState represents the local state tracking for genesys
type LocalState struct {
	Resources []ResourceRecord `json:"resources"`
}

// ResourceRecord represents a created resource
type ResourceRecord struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Type       string            `json:"type"` // "ec2", "s3", etc.
	Region     string            `json:"region"`
	Provider   string            `json:"provider"`
	ConfigFile string            `json:"config_file"`
	CreatedAt  time.Time         `json:"created_at"`
	Tags       map[string]string `json:"tags,omitempty"`
}

const stateFileName = ".genesys-state.json"

// getStateFilePath returns the path to the state file
func getStateFilePath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, stateFileName)
}

// LoadLocalState loads the local state from disk
func LoadLocalState() (*LocalState, error) {
	statePath := getStateFilePath()

	// If file doesn't exist, return empty state
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		return &LocalState{Resources: []ResourceRecord{}}, nil
	}

	data, err := os.ReadFile(statePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state LocalState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}

	return &state, nil
}

// SaveLocalState saves the local state to disk
func (s *LocalState) SaveLocalState() error {
	statePath := getStateFilePath()

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	if err := os.WriteFile(statePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

// AddResource adds a new resource to the state
func (s *LocalState) AddResource(record ResourceRecord) error {
	s.Resources = append(s.Resources, record)
	return s.SaveLocalState()
}

// RemoveResource removes a resource from the state by ID
func (s *LocalState) RemoveResource(id string) error {
	for i, resource := range s.Resources {
		if resource.ID == id {
			s.Resources = append(s.Resources[:i], s.Resources[i+1:]...)
			break
		}
	}
	return s.SaveLocalState()
}

// FindResourcesByName finds all resources with the given name
func (s *LocalState) FindResourcesByName(name string) []ResourceRecord {
	var found []ResourceRecord
	for _, resource := range s.Resources {
		if resource.Name == name {
			found = append(found, resource)
		}
	}
	return found
}

// FindResourcesByConfigFile finds all resources created from a config file
func (s *LocalState) FindResourcesByConfigFile(configFile string) []ResourceRecord {
	var found []ResourceRecord
	for _, resource := range s.Resources {
		if resource.ConfigFile == configFile {
			found = append(found, resource)
		}
	}
	return found
}

// RefreshLocalState reloads the state from disk
func RefreshLocalState() (*LocalState, error) {
	return LoadLocalState()
}

// SyncWithRemote syncs local state with remote state backend (placeholder for future implementation)
func (s *LocalState) SyncWithRemote() error {
	// TODO: Implement remote state synchronization
	// For now, just reload from disk
	newState, err := LoadLocalState()
	if err != nil {
		return fmt.Errorf("failed to refresh local state: %w", err)
	}

	s.Resources = newState.Resources
	return nil
}

// ValidateResources checks if tracked resources still exist
func (s *LocalState) ValidateResources() ([]ResourceRecord, error) {
	// TODO: Implement actual resource validation against cloud providers
	// For now, return all resources as valid
	return s.Resources, nil
}
