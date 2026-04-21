package main

import (
	"fmt"
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
	notes        []Note
	cursor       int
	showEditor   bool
	isNewNote    bool
	isRenaming   bool
	isViewing    bool   // Modo visualização (renderizado)
	isEditing   bool   // Modo edição
	showLinks   bool
	linkIndex   int

	// Links da nota atual no modo visualização
	currentLinks []string
	currentLinkIndex int

	editor      textarea.Model
	titleInput  textinput.Model
	linkList   list.Model
	currentDir string
	currentNote *Note

	// Histórico de navegação
	history    []string //lista de títulos visitados
	historyPos int      // posição atual no histórico
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
		//global quit - funciona em qualquer modo
		if msg.String() == "q" {
			return m, tea.Quit
		}
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

	// Se está no modo visualização, não permite editar
	if m.isViewing && !m.isEditing {
		keyMsg, ok := msg.(tea.KeyMsg)
		if !ok {
			return m, nil
		}
		switch keyMsg.String() {
		case "enter":
			// Navega para o link selecionado
			if len(m.currentLinks) > 0 && m.currentLinkIndex < len(m.currentLinks) {
				linkTitle := m.currentLinks[m.currentLinkIndex]
				if note := m.findNoteByTitle(linkTitle); note != nil {
					m.openNote(*note)
					return m, nil
				}
			}
			return m, nil

		case "tab":
			// Próximo link
			if len(m.currentLinks) > 0 {
				m.currentLinkIndex = (m.currentLinkIndex + 1) % len(m.currentLinks)
			}
			return m, nil

		case "shift+tab":
			// Link anterior
			if len(m.currentLinks) > 0 {
				m.currentLinkIndex--
				if m.currentLinkIndex < 0 {
					m.currentLinkIndex = len(m.currentLinks) - 1
				}
			}
			return m, nil

		case "e":
			m.isEditing = true
			m.editor.Focus()
			return m, nil

		case "alt+left":
			// Volta no histórico
			m.goBack()
			return m, nil

		case "alt+right":
			// Avança no histórico
			m.goForward()
			return m, nil

		case "ctrl+c", "esc":
			m.showEditor = false
			return m, nil
		}
		return m, nil
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
	case "enter":
		if m.isRenaming && m.isNewNote {
			// Ao pressionar Enter no campo de título de nova nota, muda para o editor
			m.isRenaming = false
			m.editor.Focus()
			return m, nil
		}
	}

	switch keyMsg.String() {
	case "ctrl+c", "esc":
		m.showLinks = false
		if m.isEditing {
			// Se estiver editando, volta para visualização
			m.isEditing = false
			m.isViewing = true
			m.isRenaming = false
			return m, nil
		}
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
		m.isEditing = false
		m.isViewing = true
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
		if m.cursor < len(m.notes) && len(m.notes) > 0 {
			m.openNote(m.notes[m.cursor])
		} else if len(m.notes) == 0 {
			m.createNewNote()
		}

	case "n":
		m.createNewNote()

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}

	case "down", "j":
		if m.cursor < len(m.notes)-1 {
			m.cursor++
		}
	}

	return m, nil
}

func (m *model) openNote(note Note) {
	m.showEditor = true
	m.isNewNote = false
	m.isViewing = true // Começa no modo visualização
	m.isEditing = false
	m.isRenaming = false
	m.currentNote = &note
	m.editor.SetValue(note.Body)
	m.titleInput.SetValue(note.Title)
	// Extrai os links da nota
	m.currentLinks = extractLinks(note.Body)
	m.currentLinkIndex = 0
	// Adiciona ao histórico
	m.pushHistory(note.Title)
}

func (m *model) createNewNote() {
	m.showEditor = true
	m.isNewNote = true
	m.isViewing = false
	m.isEditing = true
	m.isRenaming = true
	m.editor.SetValue("")
	m.titleInput.SetValue("")
	m.currentNote = nil
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
	b.WriteString(dimStyle.Render("─────────────────") + "\n")

	if len(m.notes) == 0 {
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("  nenhuma nota ainda"))
		b.WriteString(dimStyle.Render("\n  pressione n para criar"))
	} else {
		for i, note := range m.notes {
			if m.cursor == i {
				b.WriteString(focusedStyle.Render("▶ " + note.Title))
			} else {
				b.WriteString(normalStyle.Render("  " + note.Title))
			}
			b.WriteString("\n")
		}
	}

	b.WriteString(dimStyle.Render("─────────────────") + "\n")
	b.WriteString(dimStyle.Render("n = nova | o = abrir | q = sair"))

	return b.String()
}

func (m model) viewEditor() string {
	var b strings.Builder

	if m.isViewing && !m.isEditing {
		// Modo visualização (preview)
		b.WriteString(titleStyle.Render("👁 "+m.titleInput.Value()) + "\n")
		b.WriteString(dimStyle.Render("─────────────────────────") + "\n")
		b.WriteString(renderMarkdown(m.editor.Value()))
		// Mostra link selecionado
		if len(m.currentLinks) > 0 {
			b.WriteString("\n")
			b.WriteString(focusedStyle.Render("link: "+m.currentLinks[m.currentLinkIndex]))
			b.WriteString(dimStyle.Render(" ("+fmt.Sprintf("%d/%d", m.currentLinkIndex+1, len(m.currentLinks))+")"))
		}
		b.WriteString("\n" + dimStyle.Render("e = editar | Enter = abrir | Tab/Shift+Tab = navegar | Alt+←/→ = histórico | q = sair"))
	} else {
		// Modo edição
		b.WriteString(titleStyle.Render("📝 Editando nota") + "\n")
		b.WriteString(dimStyle.Render("─────────────────────────") + "\n")

		if m.isRenaming {
			b.WriteString(focusedStyle.Render("Nome: "))
			b.WriteString(m.titleInput.View())
			b.WriteString("\n")
		} else {
			b.WriteString(dimStyle.Render("Nome: " + m.titleInput.Value()))
			b.WriteString(dimStyle.Render(" (Ctrl+R)"))
			b.WriteString("\n")
		}

		b.WriteString("\n")
		b.WriteString(m.editor.View())

		b.WriteString("\n\n")
		b.WriteString(dimStyle.Render("Ctrl+S = salvar | Ctrl+R = renomear | Ctrl+L = link | q = sair"))
	}

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
	currentTitle := m.titleInput.Value()

	for _, note := range m.notes {
		// Pula a nota atual
		if note.Title == currentTitle {
			continue
		}
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

// renderMarkdown faz um preview básico do markdown para o terminal
func renderMarkdown(text string) string {
	lines := strings.Split(text, "\n")
	var b strings.Builder

	// Estilos para markdown
	h1Style := lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Bold(true).Underline(true)
	h2Style := lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
	h3Style := lipgloss.NewStyle().Foreground(lipgloss.Color("219")).Bold(true)
	boldStyle := lipgloss.NewStyle().Bold(true)
	italicStyle := lipgloss.NewStyle().Italic(true)
	codeStyle := lipgloss.NewStyle().Background(lipgloss.Color("236")).Foreground(lipgloss.Color("223"))
	externalLinkStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("81")).Underline(true)
	wikiLinkStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true).Underline(true)
	listStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("212"))

	// Expressões regulares
	h1Re := regexp.MustCompile(`^# (.+)$`)
	h2Re := regexp.MustCompile(`^## (.+)$`)
	h3Re := regexp.MustCompile(`^### (.+)$`)
	boldRe := regexp.MustCompile(`\*\*(.+?)\*\*`)
	italicRe := regexp.MustCompile(`\*(.+?)\*`)
	codeRe := regexp.MustCompile("`([^`]+)`")
	externalLinkRe := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	wikiLinkRe := regexp.MustCompile(`\[\[([^\]]+)\]\]`)
	listRe := regexp.MustCompile(`^[-*] (.+)$`)
	checkRe := regexp.MustCompile(`^- \[x\] (.+)$`)
	uncheckedRe := regexp.MustCompile(`^- \[ \] (.+)$`)

	for _, line := range lines {
		rendered := line

		// Cabeçalhos
		if match := h1Re.FindStringSubmatch(line); len(match) > 0 {
			rendered = h1Style.Render(match[1])
		} else if match := h2Re.FindStringSubmatch(line); len(match) > 0 {
			rendered = h2Style.Render(match[1])
		} else if match := h3Re.FindStringSubmatch(line); len(match) > 0 {
			rendered = h3Style.Render(match[1])
		} else {
			// Wiki links [[note]]
			if len(line) > 0 {
				matches := wikiLinkRe.FindAllStringSubmatchIndex(line, -1)
				for _, match := range matches {
					if len(match) == 4 && match[2] < match[3] && match[3] <= len(line) {
						linkText := line[match[2]:match[3]]
						rendered = strings.Replace(rendered, "[["+linkText+"]]", wikiLinkStyle.Render("‣ "+linkText), 1)
					}
				}
			}
			// Links externos [text](url) - processa manualmente
			externalLinkMatches := externalLinkRe.FindAllStringSubmatchIndex(line, -1)
			for _, match := range externalLinkMatches {
				if len(match) >= 4 {
					textStart := match[2]
					textEnd := match[3]
					if textStart >= 0 && textEnd <= len(rendered) && textStart < textEnd {
						linkText := rendered[textStart:textEnd]
						fullMatch := rendered[match[0]:match[1]]
						replacement := externalLinkStyle.Render(linkText)
						rendered = strings.Replace(rendered, fullMatch, replacement, 1)
					}
				}
			}
			// Código inline
			rendered = codeRe.ReplaceAllString(rendered, codeStyle.Render("$1"))
			// Negrito
			rendered = boldRe.ReplaceAllString(rendered, boldStyle.Render("$1"))
			// Itálico
			rendered = italicRe.ReplaceAllString(rendered, italicStyle.Render("$1"))
			// Listas
			if match := checkRe.FindStringSubmatch(line); len(match) > 0 {
				rendered = listStyle.Render("☑ " + match[1])
			} else if match := uncheckedRe.FindStringSubmatch(line); len(match) > 0 {
				rendered = listStyle.Render("☐ " + match[1])
			} else if match := listRe.FindStringSubmatch(line); len(match) > 0 {
				rendered = listStyle.Render("• " + match[1])
			}
		}

		b.WriteString(rendered)
		b.WriteString("\n")
	}

	return b.String()
}

// findNoteByTitle busca uma nota pelo título
func (m *model) findNoteByTitle(title string) *Note {
	// Recarrega notas do disco para ter certeza
	m.notes = loadNotes(m.currentDir)
	for i := range m.notes {
		if m.notes[i].Title == title {
			return &m.notes[i]
		}
	}
	return nil
}

// pushHistory adiciona uma nota ao histórico
func (m *model) pushHistory(title string) {
	if title == "" {
		return
	}
	// Protege contra índices inválidos
	if m.historyPos < 0 {
		m.historyPos = 0
		m.history = nil
	}
	// Remove o que está à frente da posição atual se houver
	if m.historyPos+1 <= len(m.history) {
		m.history = m.history[:m.historyPos+1]
	}
	// Não adiciona duplicatas consecutivas
	if len(m.history) == 0 || m.history[len(m.history)-1] != title {
		m.history = append(m.history, title)
	}
	m.historyPos = len(m.history) - 1
}

// goBack volta no histórico
func (m *model) goBack() bool {
	if m.historyPos > 0 && m.historyPos < len(m.history) {
		m.historyPos--
		title := m.history[m.historyPos]
		if note := m.findNoteByTitle(title); note != nil {
			m.currentNote = note
			m.editor.SetValue(note.Body)
			m.titleInput.SetValue(note.Title)
			return true
		}
	}
	return false
}

// goForward avança no histórico
func (m *model) goForward() bool {
	if m.historyPos < len(m.history)-1 && m.historyPos >= 0 && len(m.history) > 0 {
		m.historyPos++
		title := m.history[m.historyPos]
		if note := m.findNoteByTitle(title); note != nil {
			m.currentNote = note
			m.editor.SetValue(note.Body)
			m.titleInput.SetValue(note.Title)
			return true
		}
	}
	return false
}

// extractLinks extrai todos os títulos das notas linkadas no texto
func extractLinks(text string) []string {
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

func main() {
	p := tea.NewProgram(initialModel())
	if err := p.Start(); err != nil {
		os.Stderr.WriteString("erro: " + err.Error())
	}
}
