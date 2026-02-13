package components

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/wexinc/ralph/internal/agent"
)

func TestNewModelPicker(t *testing.T) {
	p := NewModelPicker()
	if p == nil {
		t.Fatal("expected non-nil ModelPicker")
	}
	if p.IsVisible() {
		t.Error("expected picker to not be visible initially")
	}
	if len(p.models) != 0 {
		t.Errorf("expected empty models, got %d", len(p.models))
	}
}

func TestModelPicker_SetModels(t *testing.T) {
	p := NewModelPicker()
	models := []agent.Model{
		{ID: "model-1", Name: "Model One"},
		{ID: "model-2", Name: "Model Two", IsDefault: true},
		{ID: "model-3", Name: "Model Three"},
	}

	p.SetModels(models)

	if len(p.models) != 3 {
		t.Errorf("expected 3 models, got %d", len(p.models))
	}
	// Should select default model
	if p.selected != 1 {
		t.Errorf("expected selected to be 1 (default), got %d", p.selected)
	}
}

func TestModelPicker_SetCurrentModel(t *testing.T) {
	p := NewModelPicker()
	models := []agent.Model{
		{ID: "model-1", Name: "Model One"},
		{ID: "model-2", Name: "Model Two"},
		{ID: "model-3", Name: "Model Three"},
	}
	p.SetModels(models)

	p.SetCurrentModel("model-3")

	if p.CurrentModel() != "model-3" {
		t.Errorf("expected current model 'model-3', got '%s'", p.CurrentModel())
	}
	if p.selected != 2 {
		t.Errorf("expected selected to be 2, got %d", p.selected)
	}
}

func TestModelPicker_Visibility(t *testing.T) {
	p := NewModelPicker()

	if p.IsVisible() {
		t.Error("expected not visible initially")
	}

	p.Show()
	if !p.IsVisible() {
		t.Error("expected visible after Show")
	}

	p.Hide()
	if p.IsVisible() {
		t.Error("expected not visible after Hide")
	}

	p.Toggle()
	if !p.IsVisible() {
		t.Error("expected visible after Toggle")
	}

	p.Toggle()
	if p.IsVisible() {
		t.Error("expected not visible after second Toggle")
	}
}

func TestModelPicker_Navigation(t *testing.T) {
	p := NewModelPicker()
	models := []agent.Model{
		{ID: "model-1", Name: "Model One"},
		{ID: "model-2", Name: "Model Two"},
		{ID: "model-3", Name: "Model Three"},
	}
	p.SetModels(models)

	// Initial selection
	if p.selected != 0 {
		t.Errorf("expected initial selection 0, got %d", p.selected)
	}

	// Move down
	p.MoveDown()
	if p.selected != 1 {
		t.Errorf("expected selection 1 after MoveDown, got %d", p.selected)
	}

	p.MoveDown()
	if p.selected != 2 {
		t.Errorf("expected selection 2 after MoveDown, got %d", p.selected)
	}

	// Should not go past end
	p.MoveDown()
	if p.selected != 2 {
		t.Errorf("expected selection 2 (no change), got %d", p.selected)
	}

	// Move up
	p.MoveUp()
	if p.selected != 1 {
		t.Errorf("expected selection 1 after MoveUp, got %d", p.selected)
	}

	// Move to top
	p.MoveUp()
	p.MoveUp() // Should not go past start
	if p.selected != 0 {
		t.Errorf("expected selection 0, got %d", p.selected)
	}
}

func TestModelPicker_SelectedModel(t *testing.T) {
	p := NewModelPicker()

	// Empty models
	if p.SelectedModel() != nil {
		t.Error("expected nil SelectedModel with no models")
	}

	models := []agent.Model{
		{ID: "model-1", Name: "Model One"},
		{ID: "model-2", Name: "Model Two"},
	}
	p.SetModels(models)

	m := p.SelectedModel()
	if m == nil {
		t.Fatal("expected non-nil SelectedModel")
	}
	if m.ID != "model-1" {
		t.Errorf("expected model-1, got %s", m.ID)
	}
}

func TestModelPicker_Update_NotVisible(t *testing.T) {
	p := NewModelPicker()

	cmd := p.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if cmd != nil {
		t.Error("expected nil command when not visible")
	}
}

func TestModelPicker_Update_EnterSelects(t *testing.T) {
	p := NewModelPicker()
	models := []agent.Model{
		{ID: "model-1", Name: "Model One"},
		{ID: "model-2", Name: "Model Two"},
	}
	p.SetModels(models)
	p.Show()

	p.MoveDown() // Select model-2

	cmd := p.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Should have hidden
	if p.IsVisible() {
		t.Error("expected picker to be hidden after selection")
	}

	// Should have updated current model
	if p.CurrentModel() != "model-2" {
		t.Errorf("expected current model 'model-2', got '%s'", p.CurrentModel())
	}

	// Should return message
	if cmd == nil {
		t.Fatal("expected command to be returned")
	}
	msg := cmd()
	selMsg, ok := msg.(ModelSelectedMsg)
	if !ok {
		t.Fatalf("expected ModelSelectedMsg, got %T", msg)
	}
	if selMsg.Model.ID != "model-2" {
		t.Errorf("expected selected model ID 'model-2', got '%s'", selMsg.Model.ID)
	}
}

func TestModelPicker_Update_EscCloses(t *testing.T) {
	p := NewModelPicker()
	p.Show()

	cmd := p.Update(tea.KeyMsg{Type: tea.KeyEscape})

	if p.IsVisible() {
		t.Error("expected picker to be hidden after Esc")
	}

	if cmd == nil {
		t.Fatal("expected command to be returned")
	}
	msg := cmd()
	if _, ok := msg.(ModelPickerClosedMsg); !ok {
		t.Errorf("expected ModelPickerClosedMsg, got %T", msg)
	}
}

func TestModelPicker_View_NotVisible(t *testing.T) {
	p := NewModelPicker()

	view := p.View()

	if view != "" {
		t.Errorf("expected empty view when not visible, got: %s", view)
	}
}

func TestModelPicker_View_ShowsModels(t *testing.T) {
	p := NewModelPicker()
	models := []agent.Model{
		{ID: "model-1", Name: "Model One"},
		{ID: "model-2", Name: "Model Two"},
	}
	p.SetModels(models)
	p.SetCurrentModel("model-1")
	p.Show()

	view := p.View()

	if !strings.Contains(view, "Select Model") {
		t.Error("expected view to contain 'Select Model'")
	}
	if !strings.Contains(view, "Model One") {
		t.Error("expected view to contain 'Model One'")
	}
	if !strings.Contains(view, "Model Two") {
		t.Error("expected view to contain 'Model Two'")
	}
	if !strings.Contains(view, "current") {
		t.Error("expected view to contain current indicator")
	}
}

func TestModelPicker_View_EmptyModels(t *testing.T) {
	p := NewModelPicker()
	p.Show()

	view := p.View()

	if !strings.Contains(view, "No models available") {
		t.Error("expected view to contain 'No models available'")
	}
}

func TestModelPicker_SetSize(t *testing.T) {
	p := NewModelPicker()
	p.SetSize(100, 20)

	if p.width != 100 {
		t.Errorf("expected width 100, got %d", p.width)
	}
	if p.height != 20 {
		t.Errorf("expected height 20, got %d", p.height)
	}
}

