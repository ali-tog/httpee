package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type KeyBindings struct {
	Quit            []string `json:"quit"`
	SwitchFocus     []string `json:"switch_focus"`
	Execute         []string `json:"execute"`
	MoveUp          []string `json:"move_up"`
	MoveDown        []string `json:"move_down"`
	Preview         []string `json:"preview"`
	ToggleVariables []string `json:"toggle_variables"`
	ToggleHistory   []string `json:"toggle_history"`
	CloseRightPanel []string `json:"close_right_panel"`
	ToggleHeaders   []string `json:"toggle_headers"`
	CopyBody        []string `json:"copy_body"`
	CopyCurl        []string `json:"copy_curl"`
	EditSave        []string `json:"edit_save"`
	EditCancel      []string `json:"edit_cancel"`
}

type Config struct {
	Keys KeyBindings `json:"keys"`
}

func DefaultConfig() Config {
	return Config{
		Keys: KeyBindings{
			Quit:            []string{"ctrl+c"},
			SwitchFocus:     []string{"tab"},
			Execute:         []string{"enter"},
			MoveUp:          []string{"up", "k", "pgup"},
			MoveDown:        []string{"down", "j", "pgdown"},
			Preview:         []string{"?"},
			ToggleVariables: []string{"v"},
			ToggleHistory:   []string{"ctrl+h"},
			CloseRightPanel: []string{"q", "x"},
			ToggleHeaders:   []string{"h"},
			CopyBody:        []string{"c"},
			CopyCurl:        []string{"e"},
			EditSave:        []string{"ctrl+s"},
			EditCancel:      []string{"esc"},
		},
	}
}

func Load() (Config, error) {
	config := DefaultConfig()

	home, err := os.UserHomeDir()
	if err != nil {
		return config, err
	}

	configDir := filepath.Join(home, ".httpee")
	configPath := filepath.Join(configDir, "config.json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Create directory if it does not exist
			_ = os.MkdirAll(configDir, 0755)
			
			// Try to automatically generate default config at the expected path
			if defaultData, err := json.MarshalIndent(config, "", "  "); err == nil {
				_ = os.WriteFile(configPath, defaultData, 0644)
			}
			return config, nil
		}
		return config, err
	}

	if err := json.Unmarshal(data, &config); err != nil {
		// Even if error parsing, return defaults so program doesn't crash completely.
		return config, err
	}

	return config, nil
}
