package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/list"
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

type StarterFile struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

type CodeData struct {
	Readme       string        `json:"Readme"`
	ProgLang     string        `json:"ProgLang"`
	StarterFiles []StarterFile `json:"StarterFiles"`
}

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
	LessonDataCodeTests      CodeData           `json:"LessonDataCodeTests"`
}

type Response struct {
	Lesson struct {
		UUID                string `json:"UUID"`
		Slug                string `json:"Slug"`
		Type                string `json:"Type"`
		CourseUUID          string `json:"CourseUUID"`
		CourseTitle         string `json:"CourseTitle"`
		CourseSlug          string `json:"CourseSlug"`
		ChapterUUID         string `json:"ChapterUUID"`
		ChapterTitle        string `json:"ChapterTitle"`
		ChapterSlug         string `json:"ChapterSlug"`
		LessonDataCodeTests struct {
			StarterFiles []struct {
				Name       string `json:"Name"`
				Content    string `json:"Content"`
				IsHidden   bool   `json:"IsHidden"`
				IsReadOnly bool   `json:"IsReadOnly"`
			} `json:"StarterFiles"`
			Readme string `json:"Readme"`
		} `json:"LessonDataCodeTests"`
		LessonDataCodeCompletion struct {
			StarterFiles []struct {
				Name       string `json:"Name"`
				Content    string `json:"Content"`
				IsHidden   bool   `json:"IsHidden"`
				IsReadOnly bool   `json:"IsReadOnly"`
			} `json:"StarterFiles"`
			Readme string `json:"Readme"`
		} `json:"LessonDataCodeCompletion"`
		LessonDataMultipleChoice struct {
			Question struct {
				Question string   `json:"Question"`
				Answers  []string `json:"Answers"`
				Answer   string   `json:"Answer"`
			} `json:"Question"`
		} `json:"LessonDataMultipleChoice"`
	} `json:"Lesson"`
}

type errMsg struct {
	err error
}

func (e errMsg) Error() string { return e.err.Error() }

type item struct {
	title string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return "" }
func (i item) FilterValue() string { return i.title }

// Model for our Bubble Tea application
type Model struct {
	state          int
	list           list.Model
	selectedAnswer int
	width          int
	height         int
	lessonURL      string
	err            error
	response       *Response
	attempts       int
}

func convertToAPIURL(inputURL string) string {
	// Extract UUID from the URL
	parts := strings.Split(inputURL, "/")
	var uuid string
	for i, part := range parts {
		if i == len(parts)-1 {
			uuid = part
			break
		}
	}

	// Return API URL regardless of UUID length
	return fmt.Sprintf("https://api.boot.dev/v1/static/lessons/%s", uuid)
}

func initialModel(url string) Model {
	return Model{
		list:           list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0),
		selectedAnswer: 0,
		state:          1, // Start in fetch state
		lessonURL:      convertToAPIURL(url),
		attempts:       0,
	}
}

func (m Model) Init() tea.Cmd {
	return m.fetchLesson
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.state == 2 {
				m.list.CursorUp()
			}
			return m, nil
		case "down", "j":
			if m.state == 2 {
				m.list.CursorDown()
			}
			return m, nil
		case "enter":
			if m.state == 2 {
				selectedAnswer := m.list.Items()[m.list.Index()].(item).title
				if selectedAnswer == m.response.Lesson.LessonDataMultipleChoice.Question.Answer {
					m.state = 3 // Success state
				} else {
					m.attempts++
					if m.attempts >= 3 {
						m.state = 5 // Failed state
					} else {
						m.state = 4 // Try again state
					}
				}
				return m, nil
			} else if m.state == 4 {
				// Reset to question state for retry
				m.state = 2
				return m, nil
			}
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case errMsg:
		m.err = msg.err
		return m, tea.Quit
	case *Response:
		// fmt.Println("📦 Processing response...")
		m.response = msg

		// fmt.Printf("Checking lesson type: %s\n", m.response.Lesson.Type)

		// Check if it's an MCQ lesson
		if m.response.Lesson.Type == "type_multiple_choice" || m.response.Lesson.Type == "type_choice" {
			// fmt.Println("📝 Setting up MCQ interface...")
			items := make([]list.Item, len(m.response.Lesson.LessonDataMultipleChoice.Question.Answers))
			for i, answer := range m.response.Lesson.LessonDataMultipleChoice.Question.Answers {
				items[i] = item{title: answer}
			}

			// Create a simple list just to handle the items
			m.list = list.New(items, list.NewDefaultDelegate(), 0, 0)
			m.state = 2
			return m, nil
		} else if m.response.Lesson.Type == "type_code_tests" || m.response.Lesson.Type == "type_code" {
			// fmt.Println("📁 Creating exercise files...")

			// Get chapter and lesson numbers from slugs
			chapterNum := getChapterNumber(m.response.Lesson.ChapterSlug)
			lessonNum := getLessonNumber(m.response.Lesson.Slug)

			fmt.Printf("📂 Chapter %d, Lesson %d\n", chapterNum, lessonNum)

			// Create chapter directory if it doesn't exist
			chapterDir := fmt.Sprintf("chapter%d", chapterNum)
			if err := os.MkdirAll(chapterDir, 0755); err != nil {
				m.err = fmt.Errorf("failed to create chapter directory: %v", err)
				return m, tea.Quit
			}

			// Create exercise directory using lesson number
			exerciseDir := fmt.Sprintf("%s/exercise%d", chapterDir, lessonNum)
			if err := os.MkdirAll(exerciseDir, 0755); err != nil {
				m.err = fmt.Errorf("failed to create exercise directory: %v", err)
				return m, tea.Quit
			}

			// Handle code lesson
			var starterFiles []struct {
				Name       string `json:"Name"`
				Content    string `json:"Content"`
				IsHidden   bool   `json:"IsHidden"`
				IsReadOnly bool   `json:"IsReadOnly"`
			}
			var readme string

			if m.response.Lesson.Type == "type_code_tests" {
				starterFiles = m.response.Lesson.LessonDataCodeTests.StarterFiles
				readme = m.response.Lesson.LessonDataCodeTests.Readme
			} else {
				starterFiles = m.response.Lesson.LessonDataCodeCompletion.StarterFiles
				readme = m.response.Lesson.LessonDataCodeCompletion.Readme
			}

			for _, file := range starterFiles {
				if file.IsHidden {
					continue // Skip hidden files
				}
				filePath := filepath.Join(exerciseDir, file.Name)
				if err := os.WriteFile(filePath, []byte(file.Content), 0644); err != nil {
					m.err = fmt.Errorf("failed to create %s: %v", filePath, err)
					return m, tea.Quit
				}
			}

			// Create README.md in the exercise directory
			readmePath := filepath.Join(exerciseDir, "README.md")
			if err := os.WriteFile(readmePath, []byte(readme), 0644); err != nil {
				m.err = fmt.Errorf("failed to create README.md: %v", err)
				return m, tea.Quit
			}

			// fmt.Printf("\n✅ Files created successfully in %s:\n", exerciseDir)
			for _, file := range starterFiles {
				if !file.IsHidden {
					// fmt.Printf("- %s\n", file.Name)
				}
			}
			// fmt.Println("- README.md")

			return m, tea.Quit
		} else {
			m.err = fmt.Errorf("unknown lesson type: %s", m.response.Lesson.Type)
			return m, tea.Quit
		}
	}

	if m.state == 2 {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m Model) View() string {
	if m.err != nil {
		return fmt.Sprintf("\n❌ Error: %v\n\nPress any key to exit...\n", m.err)
	}

	switch m.state {
	case 1:
		return "\n  🔄 Fetching lesson data...\n"
	case 2:
		var s strings.Builder
		s.WriteString("\n")

		// Write the question
		questionStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("252")).
			Width(m.width - 4).
			PaddingLeft(2)
		s.WriteString(questionStyle.Render(m.response.Lesson.LessonDataMultipleChoice.Question.Question))
		s.WriteString("\n\n") // Add space after question

		// Write each answer option with proper spacing
		items := m.list.Items()
		for i, item := range items {
			// Add vertical spacing between options
			if i > 0 {
				s.WriteString("\n")
			}

			// Style for the option
			style := lipgloss.NewStyle().
				Width(m.width - 4).
				PaddingLeft(4)

			// If this is the selected item, make it bold and colored
			if i == m.list.Index() {
				style = style.
					Bold(true).
					Foreground(lipgloss.Color("170"))
				s.WriteString("  ▶ ") // Add arrow for selected item
			} else {
				s.WriteString("    ") // Add space for unselected items
			}

			s.WriteString(style.Render(item.FilterValue()))
		}

		// Add key bindings at the bottom with some space
		s.WriteString("\n\n  (↑/↓: select • enter: submit • ctrl+c: quit)\n")

		return s.String()
	case 3:
		return "\n  ✅ Correct! Great job!\n\nPress enter to exit"
	case 4:
		return fmt.Sprintf("\n  ❌ Incorrect. Try again! (%d attempts remaining)\n\n  Press enter to retry...\n", 3-m.attempts)
	case 5:
		return fmt.Sprintf("\n  ❌ Incorrect! The correct answer was: %s\n\n", m.response.Lesson.LessonDataMultipleChoice.Question.Answer)
	default:
		return "\n"
	}
}

func getCreatedFiles(files []StarterFile) string {
	var result strings.Builder
	for _, file := range files {
		result.WriteString(fmt.Sprintf("- %s\n", file.Name))
	}
	result.WriteString("- README.md")
	return result.String()
}

func (m Model) fetchLesson() tea.Msg {
	// fmt.Printf("🔍 Fetching lesson data from: %s\n", m.lessonURL)

	// Debug: Print request details
	// fmt.Printf("Making HTTP GET request to: %s\n", m.lessonURL)

	resp, err := http.Get(m.lessonURL)
	if err != nil {
		fmt.Printf("❌ HTTP request failed: %v\n", err)
		return errMsg{err: fmt.Errorf("failed to fetch lesson: %v", err)}
	}
	defer resp.Body.Close()

	// fmt.Printf("📥 Got response from server (status: %s)\n", resp.Status)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		// fmt.Printf("❌ Failed to read response body: %v\n", err)
		return errMsg{err: fmt.Errorf("failed to read response: %v", err)}
	}

	// Debug: Print raw response
	// fmt.Printf("Raw response: %s\n", string(body))

	var response Response
	if err := json.Unmarshal(body, &response); err != nil {
		fmt.Printf("❌ Failed to parse JSON: %v\n", err)
		return errMsg{err: fmt.Errorf("failed to parse response: %v", err)}
	}

	// Debug: Print parsed response
	// fmt.Printf("Parsed response:\n")
	// fmt.Printf("- Type: %s\n", response.Lesson.Type)
	// fmt.Printf("- MCQ Answers: %v\n", response.Lesson.LessonDataMultipleChoice.Question.Answers)
	// fmt.Printf("- Code Files: %v\n", len(response.Lesson.LessonDataCodeTests.StarterFiles))

	return &response
}

// Extract chapter number from ChapterSlug (format: "7-advanced-pointers" -> 7)
func getChapterNumber(slug string) int {
	parts := strings.Split(slug, "-")
	if len(parts) > 0 {
		if num, err := strconv.Atoi(parts[0]); err == nil {
			return num
		}
	}
	return 0
}

// Extract lesson number from Slug (format: "2-pointer-array" -> 2)
func getLessonNumber(slug string) int {
	parts := strings.Split(slug, "-")
	if len(parts) > 0 {
		if num, err := strconv.Atoi(parts[0]); err == nil {
			return num
		}
	}
	return 0
}

func openFiles(editor string, files []string) error {
	if editor == "" {
		return nil
	}
	cmd := exec.Command(editor, files...)
	return cmd.Start()
}

func main() {
	var codeEditor string
	var mdEditor string

	flag.StringVar(&codeEditor, "code-editor", "", "Editor to open code files with (e.g., 'code', 'vim', 'emacs')")
	flag.StringVar(&mdEditor, "md-editor", "", "Editor to open markdown files with (e.g., 'typora', 'code')")
	flag.Parse()

	args := flag.Args()
	if len(args) != 1 {
		fmt.Println("Please provide the lesson URL as an argument")
		os.Exit(1)
	}

	p := tea.NewProgram(initialModel(args[0]), tea.WithAltScreen())

	model, err := p.Run()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// After program exits, check if we need to open files
	m, ok := model.(Model)
	if !ok {
		fmt.Printf("Error: unexpected model type\n")
		os.Exit(1)
	}

	if m.response != nil && (m.response.Lesson.Type == "type_code_tests" || m.response.Lesson.Type == "type_code") {
		chapterNum := getChapterNumber(m.response.Lesson.ChapterSlug)
		lessonNum := getLessonNumber(m.response.Lesson.Slug)
		exerciseDir := fmt.Sprintf("chapter%d/exercise%d", chapterNum, lessonNum)

		var codeFiles []string
		var mdFiles []string

		// Walk through the exercise directory
		err := filepath.Walk(exerciseDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				if strings.HasSuffix(info.Name(), ".md") {
					mdFiles = append(mdFiles, path)
				} else {
					codeFiles = append(codeFiles, path)
				}
			}
			return nil
		})

		if err != nil {
			fmt.Printf("Error walking directory: %v\n", err)
			os.Exit(1)
		}

		// Open code files
		if len(codeFiles) > 0 && codeEditor != "" {
			if err := openFiles(codeEditor, codeFiles); err != nil {
				fmt.Printf("Error opening code files with %s: %v\n", codeEditor, err)
			}
		}

		// Open markdown files
		if len(mdFiles) > 0 && mdEditor != "" {
			if err := openFiles(mdEditor, mdFiles); err != nil {
				fmt.Printf("Error opening markdown files with %s: %v\n", mdEditor, err)
			}
		}
	}
}
