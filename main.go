package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/bluekeyes/go-gitdiff/gitdiff"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type mainModel struct {
	input      string
	files      []*gitdiff.File
	cursor     int
	fileTree   tea.Model
	diffViewer tea.Model
	width      int
	height     int
}

func newModel(input string) mainModel {
	m := mainModel{input: input}
	m.fileTree = initialFileTreeModel()
	m.diffViewer = initialDiffModel()
	return m
}

func (m mainModel) Init() tea.Cmd {
	return tea.Batch(tea.EnterAltScreen, m.fetchFileTree, m.diffViewer.Init())
}

func (m mainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				m.diffViewer, cmd = m.diffViewer.(diffModel).SetFilePatch(m.files[m.cursor])
				cmds = append(cmds, cmd)
			}
		case "down", "j":
			if m.cursor < len(m.files)-1 {
				m.cursor++
				m.diffViewer, cmd = m.diffViewer.(diffModel).SetFilePatch(m.files[m.cursor])
				cmds = append(cmds, cmd)
			}
		}

	case tea.MouseMsg:

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case fileTreeMsg:
		m.files = msg.files
		paths := make([]string, len(m.files))
		for i, f := range m.files {
			paths[i] = f.NewName
		}
		m.fileTree = m.fileTree.(ftModel).SetFiles(paths)
		m.diffViewer, cmd = m.diffViewer.(diffModel).SetFilePatch(m.files[0])
		cmds = append(cmds, cmd)

	case errMsg:
		log.Fatal(msg.err)
	}

	m.fileTree = m.fileTree.(ftModel).SetCursor(m.cursor)

	m.diffViewer, cmd = m.diffViewer.Update(msg)
	cmds = append(cmds, cmd)
	m.fileTree, cmd = m.fileTree.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

const fileTreeWidth = 25

func (m mainModel) View() string {
	ft := lipgloss.NewStyle().
		Width(fileTreeWidth).
		Height(m.height).
		Border(lipgloss.NormalBorder(), false, true, false, false).
		BorderForeground(lipgloss.Color("8")).
		Padding(0, 1).
		Render(m.fileTree.View())
	dv := lipgloss.NewStyle().MaxHeight(m.height).Width(m.width - fileTreeWidth).Render(m.diffViewer.View())
	return lipgloss.JoinHorizontal(lipgloss.Top, ft, dv)
}

func (m mainModel) fetchFileTree() tea.Msg {
	// TODO: handle error
	files, _, _ := gitdiff.Parse(strings.NewReader(m.input + "\n"))

	return fileTreeMsg{files: files}
}

type fileTreeMsg struct {
	files []*gitdiff.File
}

func main() {
	stat, err := os.Stdin.Stat()
	if err != nil {
		panic(err)
	}

	if stat.Mode()&os.ModeNamedPipe == 0 && stat.Size() == 0 {
		fmt.Println("Try piping in some text.")
		os.Exit(1)
	}

	reader := bufio.NewReader(os.Stdin)
	var b strings.Builder

	for {
		r, _, err := reader.ReadRune()
		if err != nil && err == io.EOF {
			break
		}
		_, err = b.WriteRune(r)
		if err != nil {
			fmt.Println("Error getting input:", err)
			os.Exit(1)
		}
	}

	logger, _ := tea.LogToFile("debug.log", "debug")
	defer logger.Close()
	p := tea.NewProgram(newModel(strings.TrimSpace(b.String())), tea.WithMouseAllMotion())

	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}