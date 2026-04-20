package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/list"
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
	showLinks  bool
	linkIndex  int
	editor     textarea.Model
	titleInput textinput.Model
	linkList   list.Model
	currentDir string
	width      int
	height     int
}

func initialModel() model {
	home := os.Getenv("HOME")
	notesDir := filepath.Join(home, ".gosidian")

	// Cria diretório de notas se não existir
	os.MkdirAll(notesDir, 0755)

	notes := loadNotes(notesDir)

	ta := textarea.New()
	ta.Placeholder = "Digite sua nota aqui..."
	ta.Focus()

	ti := textinput.New()
	ti.Placeholder = "Nome da nota..."
	ti.Focus()

	// Inicializa lista de autocomplete para links
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)

	return model{
		notes:      notes,
		cursor:     0,
		showEditor: false,
		isNewNote:  false,
		isRenaming: false,
		showLinks:  false,
		linkIndex:  0,
		editor:     ta,
		titleInput: ti,
		linkList:   l,
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
		// Atualiza tamanho do textarea
		m.editor.SetWidth(msg.Width - 4)
		m.editor.SetHeight(msg.Height - 12)
		// Atualiza tamanho da lista de links
		m.linkList.SetSize(msg.Width-4, 5)
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
	// Se estiver mostrando links, redireciona para a lista
	if m.showLinks {
		return m.updateLinkListInput(msg)
	}

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
		m.showLinks = false
		return m, nil

	case "ctrl+r":
		m.isRenaming = !m.isRenaming
		if m.isRenaming {
			m.titleInput.Focus()
		} else {
			m.editor.Focus()
		}
		return m, nil

	case "ctrl+l":
		// Ativa/desativa autocomplete de links
		m.showLinks = !m.showLinks
		if m.showLinks {
			m.updateLinkList("")
		}
		return m, nil

	case "enter":
		// Insere nova linha
		m.editor, _ = m.editor.Update(msg)
		return m, nil

	case "ctrl+s":
		m.saveCurrentNote()
		m.showEditor = false
		m.isRenaming = false
		m.showLinks = false
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
	b.WriteString(dimStyle.Render("Ctrl+S = salvar | Ctrl+R = renomear | Ctrl+L = link | Esc = voltar"))

	// Mostra popup de autocomplete se ativo
	if m.showLinks {
		b.WriteString("\n\n")
		b.WriteString(dimStyle.Render("─ link para nota ─"))
		b.WriteString("\n")
		b.WriteString(m.linkList.View())
	}

	return b.String()
}

// linkItem implementa list.Item para autocomplete de links
type linkItem struct {
	title string
}

func (i linkItem) FilterValue() string { return i.title }
func (i linkItem) Title() string       { return i.title }
func (i linkItem) Description() string { return "" }

// findLinks encontra todos os links [[...]] no texto
func findLinks(text string) []string {
	var links []string
	re := regexp.MustCompile(`\[\[([^\]]+)\]\]`)
	matches := re.FindAllStringSubmatch(text, -1)
	for _, m := range matches {
		if len(m) > 1 {
			links = append(links, m[1])
		}
	}
	return links
}

// extractPartialLink extrai o texto após [[ que está sendo digitado
func extractPartialLink(text string) string {
	//Procura o último [[ que não tem ]]
	lastOpen := strings.LastIndex(text, "[[")
	if lastOpen == -1 {
		return ""
	}
	// Verifica se tem ]] depois
	resto := text[lastOpen+2:]
	if strings.Contains(resto, "]]") {
		return ""
	}
	return resto
}

// updateLinkList atualiza a lista de autocomplete com notas existentes
func (m *model) updateLinkList(filter string) {
	var items []list.Item
	for _, note := range m.notes {
		if filter == "" || strings.Contains(strings.ToLower(note.Title), strings.ToLower(filter)) {
			items = append(items, linkItem{title: note.Title})
		}
	}
	m.linkList = list.New(items, list.NewDefaultDelegate(), m.width-4, 5)
	m.linkList.SetShowStatusBar(false)
	m.linkList.SetFilteringEnabled(true)
	m.linkIndex = 0
}

// updateLinkListInput gerencia input quando autocomplete está ativo
func (m model) updateLinkListInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		var cmd tea.Cmd
		m.linkList, cmd = m.linkList.Update(msg)
		return m, cmd
	}

	switch keyMsg.String() {
	case "enter":
		// Insere o link selecionado
		if item := m.linkList.SelectedItem(); item != nil {
			li := item.(linkItem)
			m.insertLink(li.title)
		}
		m.showLinks = false
		m.editor.Focus()
		return m, nil

	case "tab":
		// Próxima opção
		m.linkList.CursorDown()
		return m, nil

	case "shift+tab":
		// Opção anterior
		m.linkList.CursorUp()
		return m, nil

	case "esc", "ctrl+l":
		m.showLinks = false
		m.editor.Focus()
		return m, nil
	}

	var cmd tea.Cmd
	m.linkList, cmd = m.linkList.Update(msg)
	return m, cmd
}

// insertLink insere o link selecionado no editor
func (m *model) insertLink(selectedTitle string) {
	text := m.editor.Value()
	partial := extractPartialLink(text)

	if partial != "" {
		// Substitui o texto parcial pelo link completo
		start := strings.LastIndex(text, "[[")
		if start != -1 {
			newText := text[:start] + "[[" + selectedTitle + "]]" + text[start+len(partial)+2:]
			m.editor.SetValue(newText)
		}
	} else {
		// Insere novo link se não há parcial
		newText := text + "[[" + selectedTitle + "]]"
		m.editor.SetValue(newText)
	}
	m.showLinks = false
	m.editor.Focus()
}

func main() {
	p := tea.NewProgram(initialModel())
	if err := p.Start(); err != nil {
		os.Stderr.WriteString("erro: " + err.Error())
	}
}
