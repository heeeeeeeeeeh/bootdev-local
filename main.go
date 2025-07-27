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

	"github.com/andreyvit/diff"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Styles
var (
	paginationStyle = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	titleStyle      = lipgloss.NewStyle().
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
	infoStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Left = "┤"
		return titleStyle.BorderStyle(b)
	}()
)

type State int

const (
	Fetch State = iota
	QuestionStart
	QuestionCorrect
	QuestionRetry
	QuestionFailed
	WriteFiles
	EditorStart
	EditorFinished
	CodeTest
	CodeTestSuccess
	CodeTestFailed
	CheckOutput
	OutputSuccess
	OutputFail
	NextLesson
	TrackSelect
	CourseSelect
	CourseFinished
	ChapterSelect
	LessonSelect
	Failed
)

type StarterFile struct {
	Name       string `json:"Name"`
	Content    string `json:"Content"`
	IsHidden   bool   `json:"IsHidden"`
	IsReadOnly bool   `json:"IsReadOnly"`
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
			StarterFiles []StarterFile `json:"StarterFiles"`
			Readme       string        `json:"Readme"`
		} `json:"LessonDataCodeTests"`
		LessonDataCodeCompletion struct {
			ProgLang           string
			StarterFiles       []StarterFile `json:"StarterFiles"`
			Readme             string        `json:"Readme"`
			CodeExpectedOutput string        `json:"CodeExpectedOutput"`
		} `json:"LessonDataCodeCompletion"`
		LessonDataChoice struct {
			Readme   string `json:"Readme"`
			Question struct {
				Question string `json:"Question"`
				Answer   string `json:"Answer"`
				Answers  string `json:"Answers"`
			} `json:"Question"`
		} `json:"LessonDataChoice"`
		LessonDataMultipleChoice struct {
			Readme   string `json:"Readme"`
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

type Chapter struct {
	UUID    string   `json:"UUID"`
	Title   string   `json:"Title"`
	Lessons []Lesson `json:"Lessons"`
}

type CourseProgressResponse struct {
	CourseUUID string    `json:"CourseUUID"`
	Chapters   []Chapter `json:"Chapters"`
}

type Course struct {
	UUID            string `json:"UUID"`
	Title           string `json:"Title"`
	FirstLessonUUID string `json:"FirstLessonUUID"`
}

type TrackResponse struct {
	UUID    string   `json:"UUID"`
	Title   string   `json:"Title"`
	Slug    string   `json:"Slug"`
	Courses []Course `json:"Courses"`
}

type Track struct {
	UUID    string   `json:"UUID"`
	Title   string   `json:"Title"`
	Slug    string   `json:"Slug"`
	Courses []Course `json:"Courses"`
}

type TracksResponse []Track

const (
	BASE_API_URL        = "https://api.boot.dev/v1/"
	LESSON_URL          = BASE_API_URL + "static/lessons/"
	COURSE_URL          = BASE_API_URL + "static/courses/slug/"
	COURSE_PROGRESS_URL = BASE_API_URL + "course_progress_by_lesson/"
	TRACK_URL           = BASE_API_URL + "static/tracks/"
	TRACKS_URL          = BASE_API_URL + "static/tracks"
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

type ItemDelegate struct{}

func (d ItemDelegate) Height() int { return 1 }

func (d ItemDelegate) Spacing() int { return 0 }

func (d ItemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }

func (d ItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)

	if !ok {
		return
	}

	str := i.title

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(strs ...string) string {
			if len(strs) > 0 {
				return selectedItemStyle.Render("> " + strs[0])
			}
			return ""
		}
	}

	fmt.Fprint(w, fn(str))
}

// Model for our Bubble Tea application
type Model struct {
	ready                  bool
	viewport               viewport.Model
	title                  string
	content                string
	chapterIndex           int
	state                  State
	list                   list.Model
	selectedAnswer         int
	width                  int
	height                 int
	lessonURL              string
	courseURL              string
	courseProgressURL      string
	trackURL               string
	err                    error
	response               *Response
	courseProgressResponse *CourseProgressResponse
	tracksResponse         *TracksResponse
	trackResponse          *TrackResponse
	starterFiles           []string
	mdEditor               string
	codeEditor             string
	attempts               int
	download               bool
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
	download := false
	if !reflect.ValueOf(url).IsZero() {
		if strings.Contains(url, "courses") {
			courseURL = convertToAPIURL(COURSE_URL, url)
			download = true
		} else {
			lessonURL = convertToAPIURL(LESSON_URL, url)
		}
	} else {
		if file, err := os.ReadFile(".last"); err == nil {
			lessonURL = string(file)
		}
	}
	return Model{
		list:                   list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0),
		download:               download,
		selectedAnswer:         0,
		state:                  Fetch, // Start in fetch state
		lessonURL:              lessonURL,
		courseURL:              courseURL,
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

func index(l list.Model) int {
	if l.FilterState() == list.Unfiltered {
		return l.Index()
	} else {
		fitem := l.SelectedItem()
		for index, item := range l.Items() {
			if item == fitem {
				return index
			}
		}
	}
	return 0
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// move index in list to last option
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// TODO: add key to edit course/lesson url
		//
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "left", "h":
			if m.list.FilterState() != list.Filtering {
				switch m.state {
				case LessonSelect:
					m.state = ChapterSelect
					m.list = m.createList(m.courseProgressResponse.Chapters)
				case ChapterSelect:
					if m.trackURL == "" {
						m.state = Fetch
						cmds = append(cmds, func() tea.Msg { return request[TracksResponse](TRACKS_URL) })
					} else {
						m.state = CourseSelect
						m.list = m.createList(m.trackResponse.Courses)
					}
				case CourseSelect:
					m.state = TrackSelect
					if reflect.ValueOf(m.tracksResponse).IsZero() {
						cmds = append(cmds, func() tea.Msg { return request[TracksResponse](TRACKS_URL) })
					} else {
						m.state = TrackSelect
						m.list = m.createList(m.tracksResponse)
					}
				}
			}
		case "enter":
			if m.list.FilterState() != list.Filtering {
				switch m.state {
				case QuestionStart:
					selectedAnswer := m.list.Items()[index(m.list)].(item).title
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
				case TrackSelect:
					m.courseProgressURL = ""
					m.courseURL = ""
					m.trackURL = TRACK_URL + (*m.tracksResponse)[index(m.list)].Slug
					cmds = append(cmds, m.fetchLesson)
				case CourseSelect:
					m.courseProgressURL = COURSE_PROGRESS_URL + (*m.trackResponse).Courses[index(m.list)].FirstLessonUUID
					cmds = append(cmds, m.fetchLesson)
				case ChapterSelect:
					m.chapterIndex = index(m.list)
					m.state = LessonSelect
					m.list = m.createList(m.courseProgressResponse.Chapters[m.chapterIndex].Lessons)
				case LessonSelect:
					m.lessonURL = (LESSON_URL +
						m.courseProgressResponse.Chapters[m.chapterIndex].Lessons[index(m.list)].UUID)
					cmds = append(cmds, m.fetchLesson)
				case QuestionRetry:
					// Reset to question state for retry
					m.state = QuestionStart
				case QuestionCorrect:
					m.state = NextLesson
					cmds = append(cmds, m.getNextLesson())
				case QuestionFailed:
					m.state = NextLesson
					cmds = append(cmds, m.getNextLesson())
				case CodeTest:
					// TODO: add testing for lessons without unit tests
					cmds = append(cmds, m.testCode())
				case CodeTestFailed:
					cmds = append(cmds, m.openEditor())
				case CodeTestSuccess:
					cmds = append(cmds, m.getNextLesson())
				case CheckOutput:
					cmds = append(cmds, m.CheckOutput())
				case OutputSuccess:
					cmds = append(cmds, m.getNextLesson())
				case OutputFail:
					cmds = append(cmds, m.openEditor())
				case NextLesson:
					cmds = append(cmds, m.getNextLesson())
				case CourseFinished:
					cmds = append(cmds, tea.Quit)
				case Failed:
					cmds = append(cmds, tea.Quit)
				}
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if !m.ready {
			m.updateViewport()
		} else {
			headerHeight := lipgloss.Height(m.headerView())
			footerHeight := lipgloss.Height(m.footerView())
			verticalMarginHeight := headerHeight + footerHeight
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - verticalMarginHeight
		}

		return m, nil
	case errMsg:
		m.err = msg.err
		m.state = Failed
		return m, nil
	case *CourseProgressResponse:
		m.courseProgressResponse = msg
		m.state = ChapterSelect
		m.list = m.createList(m.courseProgressResponse.Chapters)
	case Model:
		m = msg
		switch m.state {
		case WriteFiles:
			cmds = append(cmds, m.createCodeFiles())
		case EditorStart:
			cmds = append(cmds, m.openEditor())
		case EditorFinished:
			cmds = append(cmds, m.getLessonType())
		case CodeTestFailed:
		case CodeTestSuccess:
		case NextLesson:
			cmds = append(cmds, m.getNextLesson())
		case Fetch:
			cmds = append(cmds, m.fetchLesson)
		}
	case *TracksResponse:
		m.tracksResponse = msg
		m.state = TrackSelect
		m.list = m.createList(m.tracksResponse)
	case *TrackResponse:
		m.trackResponse = msg
		m.list = m.createList(m.trackResponse.Courses)
		m.state = CourseSelect
	case *Response:
		// fmt.Println("📦 Processing response...")
		m.response.Lesson = msg.Lesson
		m.lessonURL = LESSON_URL + msg.Lesson.UUID
		os.WriteFile(".last", []byte(m.lessonURL), 0666)

		m.state = WriteFiles
		cmds = append(cmds, m.createCodeFiles())
	}

	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m Model) updateViewport() viewport.Model {
	headerHeight := lipgloss.Height(m.headerView())
	footerHeight := lipgloss.Height(m.footerView())
	verticalMarginHeight := headerHeight + footerHeight

	// Since this program is using the full size of the viewport we
	// need to wait until we've received the window dimensions before
	// we can initialize the viewport. The initial dimensions come in
	// quickly, though asynchronously, which is why we wait for them
	// here.
	m.viewport = viewport.New(m.width, m.height-verticalMarginHeight)
	m.viewport.YPosition = headerHeight
	m.viewport.SetContent(m.content)
	m.ready = true
	switch m.state {
	case ChapterSelect, CourseSelect, LessonSelect:
		m.viewport.GotoTop()
		i := index(m.list)
		m.viewport.ScrollDown(max(0, i-1))
	default:
		m.viewport.GotoBottom()
	}
	return m.viewport
}

func (m Model) View() string {
	if m.err != nil {
		return fmt.Sprintf("\n❌ Error: %v\n\nPress any key to exit...\n", m.err)
	}

	// if m.download {
	// 	if reflect.ValueOf(m.response.Lesson.UUID).IsZero() {
	// 		return fmt.Sprintf("Downloading %s", m.courseURL)
	// 	}
	// 	return fmt.Sprintf("Downloading %s", m.response.Lesson.Slug)
	// }

	switch m.state {
	case Fetch:
		return "\n  🔄 Fetching lesson data...\n"
	case QuestionStart:
		m.list.Title = m.response.Lesson.LessonDataMultipleChoice.Question.Question
		return m.list.View()
	case QuestionCorrect:
		return "\n  ✅ Correct! Great job!\n\nPress enter to continue"
	case QuestionRetry:
		return fmt.Sprintf("\n  ❌ Incorrect. Try again! (%d attempts remaining)\n\n  Press enter to retry...\n", 3-m.attempts)
	case QuestionFailed:
		return fmt.Sprintf("\n  ❌ Incorrect! The correct answer was: %s\n\n", m.response.Lesson.LessonDataMultipleChoice.Question.Answer)
	case TrackSelect:
		m.list.Title = "Select Track"
		return m.list.View()
	case CourseSelect:
		m.list.Title = "Select Course"
		return m.list.View()
	case ChapterSelect:
		m.list.Title = "Select Chapter"
		return m.list.View()
	case LessonSelect:
		m.list.Title = "Select Lesson"
		return m.list.View()
	case CodeTest:
		return "Testing work."
	case CodeTestSuccess:
		m.title = "Code Test Successful"
		return m.formatPager()
	case CodeTestFailed:
		m.title = "Code Test Failed"
		return m.formatPager()
	case CheckOutput:
		return "Checking Output"
	case OutputSuccess:
		m.title = "Output Matches"
		return m.formatPager()
	case OutputFail:
		m.title = "Output does not match"
		return m.formatPager()
	case NextLesson:
		return "Press Enter to continue to next lesson. ctrl+c: quit"
	case CourseFinished:
		return "Course Finished 🎊. Enter to exit"
	default:
		return "\n"
	}
}

func (m Model) formatPager() string {
	return strings.Join([]string{m.headerView(), m.viewport.View(), m.footerView()}, "\n")
}

func (m Model) headerView() string {
	title := titleStyle.Render(m.title)
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(title)))
	return lipgloss.JoinHorizontal(lipgloss.Center, title, line)
}

func (m Model) footerView() string {
	info := infoStyle.Render(fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100))
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(info)))
	return lipgloss.JoinHorizontal(lipgloss.Center, line, info)
}

// func getCreatedFiles(files []StarterFile) string {
// 	var result strings.Builder
// 	for _, file := range files {
// 		result.WriteString(fmt.Sprintf("- %s\n", file.Name))
// 	}
// 	result.WriteString("- README.md")
// 	return result.String()
// }

func request[T any](url string) tea.Msg {
	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("❌ HTTP request failed: %v\n", err)
		return errMsg{err: fmt.Errorf("failed to %s lesson: %v", url, err)}
	}

	if resp.StatusCode != http.StatusOK {
		return errMsg{
			err: fmt.Errorf("HTTP request failed with status: %s", resp.Status),
		}
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

	var response T
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
			if m.courseProgressURL != "" {
				return request[CourseProgressResponse](m.courseProgressURL)
			}
			if m.trackURL != "" {
				return request[TrackResponse](m.trackURL)
			}
			return request[TracksResponse](TRACKS_URL)
		}
		res := request[Response](m.courseURL)
		switch res := res.(type) {
		case *Response:
			m.response.Course = res.Course
			if m.download {
				return request[Response](LESSON_URL + m.response.Course.FirstLessonUUID)
			}
			return request[CourseProgressResponse](COURSE_PROGRESS_URL + m.response.Course.FirstLessonUUID)
		case errMsg:
			return res
		}
	}

	return request[Response](m.lessonURL)
}

var p *tea.Program

func main() {
	var codeEditor string
	var mdEditor string
	var downloadAll bool

	flag.StringVar(&codeEditor, "code-editor", "", "Editor to open code files with (e.g., 'code', 'vim', 'emacs')")
	flag.StringVar(&mdEditor, "md-editor", "", "Editor to open markdown files with (e.g., 'typora', 'code')")
	flag.BoolVar(&downloadAll, "download", false, "Download all courses")
	flag.Parse()

	args := flag.Args()

	if len(args) > 0 {
		p = tea.NewProgram(initialModel(args[0], codeEditor, mdEditor), tea.WithAltScreen(), tea.WithMouseCellMotion())
	} else {
		p = tea.NewProgram(initialModel("", codeEditor, mdEditor), tea.WithAltScreen(), tea.WithMouseCellMotion())
	}

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

func (m Model) getLessonType() tea.Cmd {
	return func() tea.Msg {
		switch m.response.Lesson.Type {
		case "type_choice":
			m.state = QuestionStart
			m.list = m.createList(m.response.Lesson.LessonDataMultipleChoice.Question.Answers)
		case "type_code_tests":
			m.state = CodeTest
			return m.testCode()()
		case "type_code":
			m.state = CheckOutput
			return m.CheckOutput()()
		}
		return m
	}
}

func (m Model) createList(titleStruct any) list.Model {
	val := reflect.ValueOf(titleStruct)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Slice {
		return list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0) // Return empty list if not a slice
	}

	titles := make([]list.Item, val.Len())

	if val.Index(0).Kind() == reflect.Struct {
		for i := 0; i < val.Len(); i++ {
			v := val.Index(i)
			titles[i] = item{title: v.FieldByName("Title").String()}
		}
	} else {
		for i := 0; i < val.Len(); i++ {
			v := val.Index(i)
			titles[i] = item{title: v.String()}
		}
	}
	// Create a simple list just to handle the items
	l := list.New(titles, ItemDelegate{}, m.width, m.height)
	l.SetShowStatusBar(false)
	l.SetShowTitle(true)
	l.SetFilteringEnabled(true)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.SetFilteringEnabled(true)
	return l
}

//	func (m Model) renderList(title string) string {
//		m.content = m.list.View()
//		m.title = title
//		m.list.SetShowTitle(false)
//		m.list.SetShowPagination(false)
//		m.list.Paginator.SetTotalPages(1)
//		m.viewport = m.updateViewport()
//		return strings.Join([]string{m.headerView(), m.viewport.View(), m.footerView()}, "\n")
//	}
func (m Model) lessonPath() string {
	chapterDir := fmt.Sprintf("%s/%s", m.response.Lesson.CourseSlug, m.response.Lesson.ChapterSlug)
	return fmt.Sprintf("%s/%s", chapterDir, m.response.Lesson.Slug)
}

func (m Model) createCodeFiles() tea.Cmd {
	return func() tea.Msg {
		fmt.Printf(
			"📂 Course %s, Chapter %s, Lesson %s\n", m.response.Lesson.CourseSlug,
			m.response.Lesson.ChapterSlug, m.response.Lesson.Slug,
		)

		// Create chapter directory if it doesn't exist
		chapterDir := fmt.Sprintf("%s/%s", m.response.Lesson.CourseSlug, m.response.Lesson.ChapterSlug)
		exerciseDir := fmt.Sprintf("%s/%s", chapterDir, m.response.Lesson.Slug)
		if err := os.MkdirAll(exerciseDir, 0755); err != nil {
			return errMsg{err: fmt.Errorf("failed to create exercise directory: %v", err)}
		}

		// Handle code lesson
		var starterFiles []StarterFile
		var readme string

		switch m.response.Lesson.Type {
		case "type_code_tests":
			starterFiles = m.response.Lesson.LessonDataCodeTests.StarterFiles
			readme = m.response.Lesson.LessonDataCodeTests.Readme
		case "type_code":
			starterFiles = m.response.Lesson.LessonDataCodeCompletion.StarterFiles
			readme = m.response.Lesson.LessonDataCodeCompletion.Readme
		case "type_choice":
			starterFiles = []StarterFile{}
			readme = m.response.Lesson.LessonDataMultipleChoice.Readme
		default:
			return errMsg{err: fmt.Errorf("unknown lesson type: %s", m.response.Lesson.Type)}
		}

		for _, file := range starterFiles {
			if file.IsHidden {
				continue // Skip hidden files
			}
			filePath := filepath.Join(exerciseDir, file.Name)
			if _, err := os.Stat(filePath); err != nil {
				if err := os.WriteFile(filePath, []byte(file.Content), 0644); err != nil {
					return errMsg{err: fmt.Errorf("failed to create %s: %v", filePath, err)}
				}
			}
			m.starterFiles = append(m.starterFiles, filePath)
		}

		// Create README.md in the exercise directory
		readmePath := filepath.Join(exerciseDir, "README.md")
		if err := os.WriteFile(readmePath, []byte(readme), 0644); err != nil {
			return errMsg{err: fmt.Errorf("failed to create README.md: %v", err)}
		}
		m.starterFiles = append(m.starterFiles, readmePath)

		if m.download {
			m.state = NextLesson
		} else {
			m.state = EditorStart
		}
		return m
	}
}

func (m Model) openEditor() tea.Cmd {
	codeEditor := m.codeEditor
	args := m.starterFiles
	if codeEditor == "" {
		codeEditor = "nvr"
		args = append(args, "-c", "Glow", "--remote-wait-silent")
	}

	cmd := exec.Command(codeEditor, args...)

	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		if err != nil {
			return errMsg{
				err: fmt.Errorf("failed to open editor: %s\nArgs: %v", err, m.starterFiles),
			}
		}
		m.state = EditorFinished
		return m
	})
}

func (m Model) testCode() tea.Cmd {
	return func() tea.Msg {
		var makeFile string
		if m.response.Lesson.Type == "type_code_tests" {
			makeFile = ".lib/" + m.response.Lesson.LessonDataCodeTests.ProgLang + "/Makefile"
		} else {
			makeFile = ".lib/" + m.response.Lesson.LessonDataCodeCompletion.ProgLang + "/Makefile"
		}
		if _, err := os.Stat(makeFile); err != nil {
			return errMsg{
				err: fmt.Errorf("could not open MakeFile: %v", err),
			}
		}
		cmd := exec.Command("make", "-f", makeFile, path.Join(m.response.Lesson.CourseSlug,
			m.response.Lesson.ChapterSlug, m.response.Lesson.Slug))

		var stderr, stdout bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err := cmd.Start()
		if err != nil {
			return errMsg{
				err: fmt.Errorf("err in starting test: %v", err),
			}
		}

		cmd.Wait()

		if cmd.ProcessState.ExitCode() != 0 {
			m.content = stdout.String() + stderr.String()
			m.state = CodeTestFailed
		} else {
			m.content = stdout.String()
			m.state = CodeTestSuccess
		}

		m.viewport = m.updateViewport()
		return m
	}
}

func (m Model) CheckOutput() tea.Cmd {
	return func() tea.Msg {
		script := ".lib/" + m.response.Lesson.LessonDataCodeCompletion.ProgLang + "/run"
		if _, err := os.Stat(script); err != nil {
			return errMsg{
				err: fmt.Errorf("script to run program does not exist: %v", err),
			}
		}
		cmd := exec.Command("bash", script, path.Join(m.response.Lesson.CourseSlug,
			m.response.Lesson.ChapterSlug, m.response.Lesson.Slug))

		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out
		err := cmd.Start()
		if err != nil {
			return errMsg{
				err: fmt.Errorf("err in starting test: %v", err),
			}
		}

		cmd.Wait()

		if cmd.ProcessState.ExitCode() != 0 {
			m.content = out.String()
			m.state = OutputFail
		}

		if out.String() == m.response.Lesson.LessonDataCodeCompletion.CodeExpectedOutput {
			m.content = out.String()
			m.state = OutputSuccess
		} else {
			m.content = diff.CharacterDiff(out.String(), m.response.Lesson.LessonDataCodeCompletion.CodeExpectedOutput)
			m.state = OutputFail
		}
		m.viewport = m.updateViewport()
		return m
	}
}

func (m Model) commitRepo() error {
	if _, err := os.Stat(".git"); err != nil {
		return errMsg{
			err: fmt.Errorf("initialize git repo"),
		}
	}

	path := m.lessonPath()
	cmd := exec.Command("git", "add", path)
	cmd.Stderr = &bytes.Buffer{}
	err := cmd.Start()
	if err != nil {
		return errMsg{
			err: fmt.Errorf("could not add %s to git repo: %v", path, err),
		}
	}
	cmd.Wait()

	if cmd.ProcessState.ExitCode() != 0 {
		return errMsg{
			err: fmt.Errorf("could not add %s to git repo: %v", path, cmd.Stderr),
		}
	}

	courseSlug := m.response.Lesson.CourseSlug
	lessonNum := m.getLessonNumber()
	chapNum := m.getChapterNumber()
	msg := fmt.Sprintf("%s - Chapter %d - Lesson %d", courseSlug, chapNum, lessonNum)
	cmd = exec.Command("git", "commit", "-m", msg)
	err = cmd.Start()
	if err != nil {
		return errMsg{
			err: fmt.Errorf("could not start git commit: \n%v\n%v", cmd.Args, err),
		}
	}
	cmd.Wait()
	if cmd.ProcessState.ExitCode() != 0 {
		return errMsg{
			err: fmt.Errorf("could not commit git repo: %v", cmd.Stderr),
		}
	}

	cmd = exec.Command("git", "push")
	err = cmd.Start()
	if err != nil {
		return errMsg{
			err: fmt.Errorf("git push could not start %v %v", cmd.Args, err),
		}
	}
	cmd.Wait()
	if cmd.ProcessState.ExitCode() != 0 {
		return errMsg{
			err: fmt.Errorf("could not push repo: %v", cmd.Stderr),
		}
	}

	return nil
}

func (m Model) getNextLesson() tea.Cmd {
	return func() tea.Msg {
		if m.courseProgressResponse == nil {
			res := request[CourseProgressResponse](COURSE_PROGRESS_URL + m.response.Lesson.UUID)
			switch prog := res.(type) {
			case *CourseProgressResponse:
				m.courseProgressResponse = prog
			case errMsg:
				return errMsg{err: prog.err}
			}
		}

		if !m.download {
			m.commitRepo()
		}
		m.starterFiles = []string{}
		currChapNum := m.getChapterNumber() - 1
		currChap := m.courseProgressResponse.Chapters[currChapNum]
		chap := currChap
		nextLessonNum := m.getLessonNumber()
		if nextLessonNum >= len(currChap.Lessons) {
			if currChapNum+1 >= len(m.courseProgressResponse.Chapters) {
				m.state = CourseFinished
				return m
			}
			chap = m.courseProgressResponse.Chapters[currChapNum+1]
			nextLessonNum = 0
		}
		m.lessonURL = LESSON_URL + chap.Lessons[nextLessonNum].UUID
		m.state = Fetch
		return m
	}
}

// Extract chapter number from ChapterSlug (format: "7-advanced-pointers" -> 7)
func (m Model) getChapterNumber() int {
	parts := strings.Split(m.response.Lesson.ChapterSlug, "-")
	if len(parts) > 0 {
		if num, err := strconv.Atoi(parts[0]); err == nil {
			return num
		}
	}
	return 0
}

// Extract lesson number from Slug (format: "2-pointer-array" -> 2)
func (m Model) getLessonNumber() int {
	parts := strings.Split(m.response.Lesson.Slug, "-")
	if len(parts) > 0 {
		if num, err := strconv.Atoi(parts[0]); err == nil {
			return num
		}
	}
	return 0
}
