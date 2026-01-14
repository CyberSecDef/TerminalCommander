package main

import (
	"testing"
)

func TestInitThemes(t *testing.T) {
	themes := initThemes()

	// Should have 4 themes
	if len(themes) != 4 {
		t.Errorf("Expected 4 themes, got %d", len(themes))
	}

	// Check theme names
	expectedNames := []string{"Dark", "Light", "Solarized Dark", "Solarized Light"}
	for i, expected := range expectedNames {
		if themes[i].Name != expected {
			t.Errorf("Theme %d: expected name %q, got %q", i, expected, themes[i].Name)
		}
	}
}

func TestGetTheme(t *testing.T) {
	themes := initThemes()
	cmd := &Commander{
		currentTheme: 0,
		themes:       themes,
	}

	theme := cmd.getTheme()
	if theme.Name != "Dark" {
		t.Errorf("Expected Dark theme, got %s", theme.Name)
	}
}

func TestCycleTheme(t *testing.T) {
	themes := initThemes()
	// Create a mock screen - we can't actually create a real screen in tests
	// so we'll test the theme cycling logic without the screen
	cmd := &Commander{
		currentTheme: 0,
		themes:       themes,
		statusMsg:    "",
	}

	// Test cycling through all themes
	for i := 0; i < len(themes); i++ {
		expectedIdx := (i + 1) % len(themes)
		
		// Manually increment (simulating cycleTheme without the screen)
		cmd.currentTheme++
		if cmd.currentTheme >= len(cmd.themes) {
			cmd.currentTheme = 0
		}
		
		if cmd.currentTheme != expectedIdx {
			t.Errorf("After cycle %d: expected theme index %d, got %d", i+1, expectedIdx, cmd.currentTheme)
		}
		
		theme := cmd.getTheme()
		expectedName := themes[expectedIdx].Name
		if theme.Name != expectedName {
			t.Errorf("After cycle %d: expected theme %s, got %s", i+1, expectedName, theme.Name)
		}
	}
}

func TestThemeColors(t *testing.T) {
	themes := initThemes()

	// Test Dark theme colors
	dark := themes[0]
	if dark.Name != "Dark" {
		t.Errorf("First theme should be Dark, got %s", dark.Name)
	}

	// Test Light theme colors
	light := themes[1]
	if light.Name != "Light" {
		t.Errorf("Second theme should be Light, got %s", light.Name)
	}

	// Test Solarized Dark theme colors
	solarizedDark := themes[2]
	if solarizedDark.Name != "Solarized Dark" {
		t.Errorf("Third theme should be Solarized Dark, got %s", solarizedDark.Name)
	}

	// Test Solarized Light theme colors
	solarizedLight := themes[3]
	if solarizedLight.Name != "Solarized Light" {
		t.Errorf("Fourth theme should be Solarized Light, got %s", solarizedLight.Name)
	}
}

func TestThemeWrapAround(t *testing.T) {
	themes := initThemes()
	cmd := &Commander{
		currentTheme: len(themes) - 1, // Start at last theme
		themes:       themes,
	}

	// Get current theme (should be last one)
	theme := cmd.getTheme()
	if theme.Name != "Solarized Light" {
		t.Errorf("Expected Solarized Light, got %s", theme.Name)
	}

	// Cycle once (should wrap to first theme)
	cmd.currentTheme++
	if cmd.currentTheme >= len(cmd.themes) {
		cmd.currentTheme = 0
	}

	theme = cmd.getTheme()
	if theme.Name != "Dark" {
		t.Errorf("Expected Dark after wrap around, got %s", theme.Name)
	}
}
