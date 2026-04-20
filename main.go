package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	focusedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	normalStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("247"))
	dimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	titleStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Bold(true)
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Background(lipgloss.Color("236"))
)

type Note struct {
	Title string
	Path  string
	Body  string
}

type model struct {
	notes      []Note
	cursor     int
	showEditor bool
	isNewNote  bool
	isRenaming bool
	editor     textarea.Model
	titleInput textinput.Model
	currentDir string
	width      int
	height     int
}

func initialModel() model {
	home := os.Getenv("HOME")
	notesDir := filepath.Join(home, "gosidian")

	// Cria diretório de notas se não existir
	os.MkdirAll(notesDir, 0755)

	notes := loadNotes(notesDir)

	ta := textarea.New()
	ta.Placeholder = "Digite sua nota aqui..."
	ta.Focus()

	ti := textinput.New()
	ti.Placeholder = "Nome da nota..."
	ti.Focus()

	return model{
		notes:      notes,
		cursor:     0,
		showEditor: false,
		isNewNote:  false,
		isRenaming: false,
		editor:     ta,
		titleInput: ti,
		currentDir: notesDir,
	}
}

func loadNotes(dir string) []Note {
	var notes []Note

	entries, err := os.ReadDir(dir)
	if err != nil {
		return notes
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		title := strings.TrimSuffix(entry.Name(), ".md")
		notes = append(notes, Note{
			Title: title,
			Path:  path,
			Body:  string(data),
		})
	}

	return notes
}

func (m model) Init() tea.Cmd {
	return textarea.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if m.showEditor {
			return m.updateEditor(msg)
		}
		return m.updateSidebar(msg)
	}

	return m, nil
}

func (m model) updateEditor(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		var cmd tea.Cmd
		if m.isRenaming {
			m.titleInput, cmd = m.titleInput.Update(msg)
		} else {
			m.editor, cmd = m.editor.Update(msg)
		}
		return m, cmd
	}

	switch keyMsg.String() {
	case "ctrl+c", "esc":
		m.showEditor = false
		m.isRenaming = false
		return m, nil

	case "ctrl+r":
		m.isRenaming = !m.isRenaming
		if m.isRenaming {
			m.titleInput.Focus()
		} else {
			m.editor.Focus()
		}
		return m, nil

	case "ctrl+s":
		m.saveCurrentNote()
		m.showEditor = false
		m.isRenaming = false
		return m, nil
	}

	var cmd tea.Cmd
	if m.isRenaming {
		m.titleInput, cmd = m.titleInput.Update(msg)
	} else {
		m.editor, cmd = m.editor.Update(msg)
	}
	return m, cmd
}

func (m model) updateSidebar(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "enter", "o":
		if m.cursor < len(m.notes) {
			m.openNote(m.notes[m.cursor])
		} else {
			m.createNewNote()
		}

	case "n":
		m.createNewNote()

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}

	case "down", "j":
		if m.cursor < len(m.notes) {
			m.cursor++
		}
	}

	return m, nil
}

func (m *model) openNote(note Note) {
	m.showEditor = true
	m.isNewNote = false
	m.isRenaming = false
	m.editor.SetValue(note.Body)
	m.titleInput.SetValue(note.Title)
}

func (m *model) createNewNote() {
	m.showEditor = true
	m.isNewNote = true
	m.isRenaming = true
	m.editor.SetValue("")
	m.titleInput.SetValue("")
	m.cursor = len(m.notes)
}

func (m *model) saveCurrentNote() {
	body := m.editor.Value()
	title := m.titleInput.Value()

	if title == "" {
		title = "nova-nota"
	}

	newPath := filepath.Join(m.currentDir, title+".md")
	oldPath := ""

	// Se não é nova nota e o título mudou, renomeia o arquivo
	if !m.isNewNote && m.cursor < len(m.notes) {
		oldPath = m.notes[m.cursor].Path
		if oldPath != newPath {
			os.Rename(oldPath, newPath)
		} else {
			newPath = oldPath
		}
	}

	os.WriteFile(newPath, []byte(body), 0644)

	m.notes = loadNotes(m.currentDir)
}

func (m model) View() string {
	if m.showEditor {
		return m.viewEditor()
	}
	return m.viewSidebar()
}

func (m model) viewSidebar() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("📁 gosidian") + "\n")
	b.WriteString(dimStyle.Render("─────────────────") + "\n\n")

	b.WriteString(normalStyle.Render("  atalhos:") + "\n")
	b.WriteString(dimStyle.Render("    n = nova nota") + "\n")
	b.WriteString(dimStyle.Render("    o = abrir nota") + "\n")
	b.WriteString(dimStyle.Render("    q = sair") + "\n\n")

	b.WriteString(dimStyle.Render("─────────────────") + "\n\n")

	if len(m.notes) == 0 {
		b.WriteString(dimStyle.Render("  nenhuma nota ainda"))
		b.WriteString(dimStyle.Render("\n  pressione n para criar"))
		return b.String()
	}

	for i, note := range m.notes {
		if m.cursor == i {
			b.WriteString(focusedStyle.Render("▶ " + note.Title))
		} else {
			b.WriteString(normalStyle.Render("  " + note.Title))
		}
		b.WriteString("\n")
	}

	return b.String()
}

func (m model) viewEditor() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("📝 Editando nota") + "\n")
	b.WriteString(dimStyle.Render("─────────────────────────") + "\n\n")

	if m.isRenaming {
		b.WriteString(focusedStyle.Render("Nome: "))
		b.WriteString(m.titleInput.View())
		b.WriteString("\n\n")
	} else {
		b.WriteString(dimStyle.Render("Nome: "+m.titleInput.Value()+" (Ctrl+R para alterar)") + "\n\n")
	}

	b.WriteString(m.editor.View())

	b.WriteString("\n\n")
	b.WriteString(dimStyle.Render("Ctrl+S = salvar | Ctrl+R = renomear | Esc = voltar"))

	return b.String()
}

func main() {
	p := tea.NewProgram(initialModel())
	if err := p.Start(); err != nil {
		os.Stderr.WriteString("erro: " + err.Error())
	}
}
