package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF69B4")).
			MarginLeft(2)

	itemStyle = lipgloss.NewStyle().
			PaddingLeft(4)

	selectedItemStyle = lipgloss.NewStyle().
				PaddingLeft(2).
				Foreground(lipgloss.Color("170"))

	correctStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00FF00"))

	incorrectStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF0000"))

	docStyle = lipgloss.NewStyle().Margin(1, 2)

	listStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, false, false).
			MarginTop(0).
			MarginBottom(0).
			PaddingTop(0).
			PaddingBottom(0)
)

type Question struct {
	Question string   `json:"Question"`
	Answers  []string `json:"Answers"`
	Answer   string   `json:"Answer"`
}

type MultipleChoiceData struct {
	Readme   string   `json:"Readme"`
	Question Question `json:"Question"`
}

type Lesson struct {
	UUID                     string             `json:"UUID"`
	Title                    string             `json:"Title"`
	LessonDataMultipleChoice MultipleChoiceData `json:"LessonDataMultipleChoice"`
}

type Response struct {
	Lesson Lesson `json:"Lesson"`
}

// Model for our Bubble Tea application
type model struct {
	state          int // 0: input, 1: processing, 2: display result, 3: show feedback
	textInput      textinput.Model
	err            error
	response       *Response
	list           list.Model
	selectedAnswer string
	isCorrect      bool
	attempts       int
	width          int
	height         int
}

type item struct {
	title string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return "" }
func (i item) FilterValue() string { return i.title }

func initialModel() model {
	ti := textinput.New()
	ti.Placeholder = "Enter lesson UUID or URL"
	ti.Focus()
	ti.CharLimit = 150
	ti.Width = 80

	// Initialize an empty list with proper styling
	delegate := list.NewDefaultDelegate()
	l := list.New([]list.Item{}, delegate, 0, 0)
	l.SetShowTitle(true)
	l.SetFilteringEnabled(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)

	return model{
		textInput: ti,
		state:     0,
		attempts:  0,
		list:      l,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		tea.EnterAltScreen,
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width
		m.list.SetSize(msg.Width-4, msg.Height-8)
		return m, nil

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyEnter:
			switch m.state {
			case 0:
				m.state = 1
				return m, m.fetchLesson
			case 2:
				if selected, ok := m.list.SelectedItem().(item); ok {
					m.selectedAnswer = selected.Title()
					m.isCorrect = m.selectedAnswer == m.response.Lesson.LessonDataMultipleChoice.Question.Answer
					m.state = 3
					m.attempts++
					if !m.isCorrect && m.attempts < 3 {
						return m, tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
							return resetMsg{}
						})
					}
				}
			}
		}
	case resetMsg:
		if !m.isCorrect && m.attempts < 3 {
			m.state = 2
		}
	case error:
		m.err = msg
		return m, nil
	case *Response:
		m.response = msg
		if len(m.response.Lesson.LessonDataMultipleChoice.Question.Answers) > 0 {
			items := make([]list.Item, len(m.response.Lesson.LessonDataMultipleChoice.Question.Answers))
			for i, answer := range m.response.Lesson.LessonDataMultipleChoice.Question.Answers {
				items[i] = item{title: answer}
			}

			// Create custom delegate with styling
			delegate := list.NewDefaultDelegate()
			normalStyle := lipgloss.NewStyle().
				PaddingLeft(2).
				MarginTop(0).
				MarginBottom(0)
			selectedStyle := lipgloss.NewStyle().
				PaddingLeft(2).
				Foreground(lipgloss.Color("170")).
				Bold(true).
				MarginTop(0).
				MarginBottom(0)

			delegate.Styles.NormalTitle = normalStyle
			delegate.Styles.SelectedTitle = selectedStyle

			m.list.SetItems(items)
			m.list.Title = m.response.Lesson.LessonDataMultipleChoice.Question.Question
			m.list.Styles.Title = titleStyle.Copy().
				MarginLeft(2).
				MarginBottom(1).
				Bold(true).
				Foreground(lipgloss.Color("#FFA500"))

			if m.width > 0 && m.height > 0 {
				m.list.SetSize(m.width-4, m.height-6) // Reduced height padding
			}

			m.state = 2
		}
		return m, nil
	}

	switch m.state {
	case 0:
		m.textInput, cmd = m.textInput.Update(msg)
	case 2:
		m.list, cmd = m.list.Update(msg)
	}

	return m, cmd
}

type resetMsg struct{}

func (m model) View() string {
	switch m.state {
	case 0:
		return docStyle.Render(fmt.Sprintf(
			"%s\n%s\n%s",
			titleStyle.Render("🎓 Boot.dev Lesson Fetcher"),
			m.textInput.View(),
			"(press esc to quit)",
		))
	case 1:
		return docStyle.Render("Fetching lesson data...")
	case 2:
		if m.err != nil {
			return docStyle.Render(fmt.Sprintf("Error: %v", m.err))
		}
		return docStyle.Render(fmt.Sprintf(
			"%s\n%s\n%s",
			titleStyle.Render("Multiple Choice Question"),
			m.list.View(),
			"(↑/↓: navigate • enter: select • esc: quit)",
		))
	case 3:
		if m.isCorrect {
			return docStyle.Render(fmt.Sprintf(
				"%s\n%s\n%s",
				titleStyle.Render("Quiz Result"),
				correctStyle.Render("✅ Correct! Well done!"),
				"Press Ctrl+C to exit",
			))
		} else if m.attempts >= 3 {
			return docStyle.Render(fmt.Sprintf(
				"%s\n%s\n%s\n%s",
				titleStyle.Render("Quiz Result"),
				incorrectStyle.Render("❌ Incorrect!"),
				fmt.Sprintf("The correct answer was: %s", m.response.Lesson.LessonDataMultipleChoice.Question.Answer),
				"Press Ctrl+C to exit",
			))
		} else {
			return docStyle.Render(fmt.Sprintf(
				"%s\n%s\n%s",
				titleStyle.Render("Quiz Result"),
				incorrectStyle.Render("❌ Incorrect! Try again!"),
				fmt.Sprintf("Attempts remaining: %d", 3-m.attempts),
			))
		}
	default:
		return docStyle.Render("Something went wrong!")
	}
}

func (m model) fetchLesson() tea.Msg {
	input := strings.TrimSpace(m.textInput.Value())

	uuid := input
	if len(input) > 36 {
		uuid = filepath.Base(input)
	}
	uuid = string([]byte(uuid)[:36])

	url := fmt.Sprintf("https://api.boot.dev/v1/static/lessons/%s", uuid)

	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	err = os.WriteFile("response.json", body, 0644)
	if err != nil {
		return err
	}

	var response Response
	err = json.Unmarshal(body, &response)
	if err != nil {
		return err
	}

	return &response
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
