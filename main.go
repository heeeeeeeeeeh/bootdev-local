package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"reflect"
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

type State int

const (
	Fetch State = iota
	QuestionStart
	QuestionCorrect
	QuestionRetry
	QuestionFailed
	CodeStart
	CodeTest
	NextLesson
	CourseFinished
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
			ProgLang     string
			StarterFiles []struct {
				Name       string `json:"Name"`
				Content    string `json:"Content"`
				IsHidden   bool   `json:"IsHidden"`
				IsReadOnly bool   `json:"IsReadOnly"`
			} `json:"StarterFiles"`
			Readme string `json:"Readme"`
		} `json:"LessonDataCodeTests"`
		LessonDataCodeCompletion struct {
			ProgLang     string
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
	Course struct {
		UUID                         string   `json:"UUID"`
		Slug                         string   `json:"Slug"`
		Title                        string   `json:"Title"`
		GenericTitle                 string   `json:"GenericTitle"`
		ShortDescription             string   `json:"ShortDescription"`
		Description                  string   `json:"Description"`
		ThumbnailURL                 string   `json:"ThumbnailURL"`
		PrerequisiteCourseUUIDS      []string `json:"PrerequisiteCourseUUIDS"`
		EstimatedCompletionTimeHours int      `json:"EstimatedCompletionTimeHours"`
		TypeDescription              string   `json:"TypeDescription"`
		LastUpdated                  string   `json:"LastUpdated"`
		SlugAliases                  []string `json:"SlugAliases"`
		AuthorUUIDs                  []string `json:"AuthorUUIDs"`
		MaintainerUUIDs              []string `json:"MaintainerUUIDs"`
		Alternatives                 []string `json:"Alternatives"`
		Status                       string   `json:"Status"`
		NumLessons                   int      `json:"NumLessons"`
		Chapters                     []struct {
			UUID        string `json:"UUID"`
			Slug        string `json:"Slug"`
			Title       string `json:"Title"`
			Description string `json:"Description"`
			Lessons     any    `json:"Lessons"`
			NumLessons  int    `json:"NumLessons"`
			CourseUUID  string `json:"CourseUUID"`
		} `json:"Chapters"`
		Language     string `json:"Language"`
		CompletionXp int    `json:"CompletionXp"`
		NumEnrolled  int    `json:"NumEnrolled"`
		Rating       struct {
			Average    float64 `json:"Average"`
			TotalCount int     `json:"TotalCount"`
		} `json:"Rating"`
		FirstLessonUUID        string `json:"FirstLessonUUID"`
		RecommendedCommunities []struct {
			Link string `json:"Link"`
			Name string `json:"Name"`
		} `json:"RecommendedCommunities"`
		Teachers struct {
			Authors []struct {
				UUID            string `json:"UUID"`
				FirstName       string `json:"FirstName"`
				LastName        string `json:"LastName"`
				Slug            string `json:"Slug"`
				Subtitle        string `json:"Subtitle"`
				Bio             string `json:"Bio"`
				YouTubeURL      string `json:"YouTubeURL"`
				TwitterURL      string `json:"TwitterURL"`
				GitHubURL       string `json:"GitHubURL"`
				LinkedInURL     string `json:"LinkedInURL"`
				ProfileImageURL string `json:"ProfileImageURL"`
				TwitchURL       string `json:"TwitchURL"`
			} `json:"Authors"`
			Maintainers []struct {
				UUID            string `json:"UUID"`
				FirstName       string `json:"FirstName"`
				LastName        string `json:"LastName"`
				Slug            string `json:"Slug"`
				Subtitle        string `json:"Subtitle"`
				Bio             string `json:"Bio"`
				YouTubeURL      string `json:"YouTubeURL"`
				TwitterURL      string `json:"TwitterURL"`
				GitHubURL       string `json:"GitHubURL"`
				LinkedInURL     string `json:"LinkedInURL"`
				ProfileImageURL string `json:"ProfileImageURL"`
				TwitchURL       string `json:"TwitchURL"`
			} `json:"Maintainers"`
		} `json:"Teachers"`
	}
}

type CourseProgressResponse struct {
	CourseUUID string `json:"CourseUUID"`
	Chapters   []struct {
		UUID    string `json:"UUID"`
		Title   string `json:"Title"`
		Lessons []struct {
			UUID       string `json:"UUID"`
			Title      string `json:"Title"`
			IsRequired bool   `json:"IsRequired"`
			IsComplete bool   `json:"IsComplete"`
			IsReset    bool   `json:"IsReset"`
		}
	}
}

const (
	BASE_API_URL        = "https://api.boot.dev/v1/"
	LESSON_URL          = BASE_API_URL + "static/lessons/"
	COURSE_URL          = BASE_API_URL + "static/courses/slug/"
	COURSE_PROGRESS_URL = BASE_API_URL + "course_progress_by_lesson/"
)

type errMsg struct {
	err error
}

func (e errMsg) Error() string { return e.err.Error() }

type item struct {
	title string
}

func (i item) Title() string { return i.title }

func (i item) Description() string { return "" }

func (i item) FilterValue() string { return i.title }

// Model for our Bubble Tea application
type Model struct {
	state                  State
	list                   list.Model
	selectedAnswer         int
	width                  int
	height                 int
	lessonURL              string
	err                    error
	courseURL              string
	response               *Response
	courseProgressResponse *CourseProgressResponse
	starterFiles           []string
	mdEditor               string
	codeEditor             string
	attempts               int
}

func convertToAPIURL(endpoint string, inputURL string) string {
	// Extract UUID from the URL
	parts := strings.Split(inputURL, "/")

	// Return API URL regardless of UUID length
	return fmt.Sprintf("%s%s", endpoint, parts[len(parts)-1])
}

func initialModel(url string, mdEditor string, codeEditor string) Model {
	var lessonURL string
	var courseURL string
	lessonURL = url
	if strings.Contains(url, "courses") {
		lessonURL = ""
		courseURL = url
	}
	return Model{
		list:                   list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0),
		selectedAnswer:         0,
		state:                  Fetch, // Start in fetch state
		lessonURL:              convertToAPIURL(LESSON_URL, lessonURL),
		courseURL:              convertToAPIURL(COURSE_URL, courseURL),
		attempts:               0,
		mdEditor:               mdEditor,
		codeEditor:             codeEditor,
		courseProgressResponse: nil,
		response:               &Response{},
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
			if m.state == QuestionStart {
				m.list.CursorUp()
			}
			return m, nil
		case "down", "j":
			if m.state == QuestionStart {
				m.list.CursorDown()
			}
			return m, nil
		case "enter":
			switch m.state {
			case QuestionStart:
				selectedAnswer := m.list.Items()[m.list.Index()].(item).title
				if selectedAnswer == m.response.Lesson.LessonDataMultipleChoice.Question.Answer {
					m.state = QuestionCorrect // Success state
				} else {
					m.attempts++
					if m.attempts >= 3 {
						m.state = QuestionFailed // Failed state
					} else {
						m.state = QuestionRetry // Try again state
					}
				}
				return m, nil
			case QuestionRetry:
				// Reset to question state for retry
				m.state = QuestionStart
				return m, nil
			case CodeStart:
				cmd, err := openFiles(m.codeEditor, m.starterFiles)
				if err != nil {
					m.err = fmt.Errorf("could not open code files %v", err)
					return m, tea.Quit
				}
				cmd2, err := openFiles(m.mdEditor, []string{"README.md"})
				if err != nil {
					m.err = fmt.Errorf("could not open code files %v", err)
					return m, tea.Quit
				}
				cmd.Wait()
				cmd2.Wait()
				if m.response.Lesson.Type == "type_code_tests" {
					m.state = CodeTest
				} else {
					m.state = NextLesson
				}
				return m, nil
			case CodeTest:
				var makeFile string
				if m.response.Lesson.Type == "type_code_tests" {
					makeFile = ".lib/" + m.response.Lesson.LessonDataCodeTests.ProgLang + "/Makefile"
				} else {
					makeFile = ".lib/" + m.response.Lesson.LessonDataCodeCompletion.ProgLang + "/Makefile"
				}
				if _, err := os.Stat(makeFile); err != nil {
					m.err = fmt.Errorf("could not open MakeFile: %v", err)
					return m, tea.Quit
				}
				cmd := exec.Command("make", "-f", makeFile, path.Join(m.response.Lesson.CourseSlug,
					m.response.Lesson.ChapterSlug, m.response.Lesson.Slug))

				var stderr, stdout bytes.Buffer
				cmd.Stdout = &stdout
				cmd.Stderr = &stderr
				err := cmd.Start()
				if err != nil {
					m.err = fmt.Errorf("err in starting test: %v", err)
				}

				cmd.Wait()

				if cmd.ProcessState.ExitCode() != 0 {
					m.err = fmt.Errorf("test failed %v", stderr.String())
					return m, tea.Quit
				}
				print(stdout.String())
				m.state = NextLesson
				return m, nil
			case NextLesson:
				m.state = Fetch
				chapNum := getChapterNumber(m.response.Lesson.ChapterSlug)
				chap := m.courseProgressResponse.Chapters[chapNum]
				lessonNum := getLessonNumber(m.response.Lesson.Slug) + 1
				if lessonNum >= len(chap.Lessons) {
					chapNum = chapNum + 1
					if chapNum >= len(m.courseProgressResponse.Chapters) {
						m.state = CourseFinished
						return m, nil
					}
					chap = m.courseProgressResponse.Chapters[chapNum]
					lessonNum = 0
				}
				m.lessonURL = chap.Lessons[lessonNum].UUID
				return m, m.fetchLesson
			case CourseFinished:
				return m, tea.Quit
			}
			return m, nil
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case errMsg:
		m.err = msg.err
		return m, tea.Quit
	case *CourseProgressResponse:
		m.err = fmt.Errorf("picking lesson from course url not implemeted yet")
		return m, tea.Quit
	case *Response:
		// fmt.Println("📦 Processing response...")
		if m.courseProgressResponse == nil {
			m.response.Lesson = msg.Lesson
			res := m.request(COURSE_PROGRESS_URL+msg.Lesson.CourseUUID, reflect.TypeOf(&CourseProgressResponse{}))
			switch prog := res.(type) {
			case *CourseProgressResponse:
				m.courseProgressResponse = prog
			case errMsg:
				m.err = prog.err
				return m, tea.Quit
			}
		}

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
			m.state = QuestionStart
			return m, nil
		} else if m.response.Lesson.Type == "type_code_tests" || m.response.Lesson.Type == "type_code" {
			// fmt.Println("📁 Creating exercise files...")

			fmt.Printf(
				"📂 Course %s, Chapter %s, Lesson %s\n", m.response.Lesson.CourseSlug,
				m.response.Lesson.ChapterSlug, m.response.Lesson.Slug,
			)

			// Create chapter directory if it doesn't exist
			chapterDir := fmt.Sprintf("%s/%s", m.response.Lesson.CourseSlug, m.response.Lesson.ChapterSlug)
			exerciseDir := fmt.Sprintf("%s/%s", chapterDir, m.response.Lesson.Slug)
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
				m.starterFiles = append(m.starterFiles, file.Name)
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

			m.state = CodeStart
			return m, nil
		} else {
			m.err = fmt.Errorf("unknown lesson type: %s", m.response.Lesson.Type)
			return m, tea.Quit
		}
	}

	if m.state == QuestionStart {
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
	case Fetch:
		return "\n  🔄 Fetching lesson data...\n"
	case QuestionStart:
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
	case QuestionCorrect:
		return "\n  ✅ Correct! Great job!\n\nPress enter to continue"
	case QuestionRetry:
		return fmt.Sprintf("\n  ❌ Incorrect. Try again! (%d attempts remaining)\n\n  Press enter to retry...\n", 3-m.attempts)
	case QuestionFailed:
		return fmt.Sprintf("\n  ❌ Incorrect! The correct answer was: %s\n\n", m.response.Lesson.LessonDataMultipleChoice.Question.Answer)
	case CodeStart:
		return "Opening files. enter to continue"
	case CodeTest:
		return "Testing work. enter to continue"
	case NextLesson:
		return "Press Enter to continue to next lesson. ctrl+c: quit"
	case CourseFinished:
		return "Course Finished 🎊. Enter to exit"
	default:
		return "\n"
	}
}

// func getCreatedFiles(files []StarterFile) string {
// 	var result strings.Builder
// 	for _, file := range files {
// 		result.WriteString(fmt.Sprintf("- %s\n", file.Name))
// 	}
// 	result.WriteString("- README.md")
// 	return result.String()
// }

func (m Model) request(url string, t reflect.Type) tea.Msg {
	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("❌ HTTP request failed: %v\n", err)
		return errMsg{err: fmt.Errorf("failed to %s lesson: %v", url, err)}
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

	response := reflect.New(t).Interface()
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

func (m Model) fetchLesson() tea.Msg {
	// fmt.Printf("🔍 Fetching lesson data from: %s\n", m.lessonURL)

	// Debug: Print request details
	// fmt.Printf("Making HTTP GET request to: %s\n", m.lessonURL)
	if m.lessonURL == "" {
		if m.courseURL == "" {
			return errMsg{
				err: fmt.Errorf("course and lesson url are both undefined"),
			}
		}
		res := m.request(m.courseURL, reflect.TypeOf(&Response{}))
		switch res := res.(type) {
		case *Response:
			m.response.Course = res.Course
			courseProgress := m.request(COURSE_PROGRESS_URL+m.response.Course.UUID, reflect.TypeOf(&CourseProgressResponse{}))
			return courseProgress
		case errMsg:
			return res
		}
	}

	return m.request(m.lessonURL, reflect.TypeOf(&Response{}))
}

func openFiles(editor string, files []string) (*exec.Cmd, error) {
	var cmd *exec.Cmd

	if editor == "" {
		for _, file := range files[1:] {
			cmd = exec.Command(editor, file)
			cmd.Stderr = os.Stderr
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			err := cmd.Start()
			if err != nil {
				return cmd, err
			}
		}
		cmd = exec.Command(editor, files[0])
	} else {
		cmd = exec.Command(editor, files...)
	}

	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	return cmd, cmd.Start()
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

	p := tea.NewProgram(initialModel(args[0], codeEditor, mdEditor), tea.WithAltScreen())

	model, err := p.Run()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// After program exits, check if we need to open files
	_, ok := model.(Model)
	if !ok {
		fmt.Printf("Error: unexpected model type\n")
		os.Exit(1)
	}
}
