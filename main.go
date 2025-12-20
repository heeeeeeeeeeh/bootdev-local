package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	"github.com/andreyvit/diff"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/thoas/go-funk"
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
	InputDir
	CLICheck
	CLIDone
	CLIFailed
	CheckInput
	InputFail
	InputSuccess
	Git
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

type CLIData struct {
	// ContainsCompleteDir bool
	BaseURLDefault string
	Steps          []CLIStep
}

type CLIStep struct {
	CLICommand  *CLIStepCLICommand
	HTTPRequest *CLIStepHTTPRequest
}

type CLIStepCLICommand struct {
	Command string
	Tests   []CLICommandTest
}

type CLICommandTest struct {
	ExitCode           *int
	StdoutContainsAll  []string
	StdoutContainsNone []string
	StdoutLinesGt      *int
}

type CLIStepHTTPRequest struct {
	ResponseVariables []HTTPRequestResponseVariable
	Tests             []HTTPRequestTest
	Request           HTTPRequest
}

type HTTPRequest struct {
	Method   string
	FullURL  string
	Headers  map[string]string
	BodyJSON map[string]any

	BasicAuth *HTTPBasicAuth
	Actions   HTTPActions
}

type HTTPBasicAuth struct {
	Username string
	Password string
}

type HTTPActions struct {
	DelayRequestByMs *int
}

type HTTPRequestTest struct {
	StatusCode       *int
	BodyContains     *string
	BodyContainsNone *string
	HeadersContain   *HTTPRequestTestHeader
	TrailersContain  *HTTPRequestTestHeader
	JSONValue        *HTTPRequestTestJSONValue
}

type HTTPRequestTestHeader struct {
	Key   string
	Value string
}

type HTTPRequestTestJSONValue struct {
	Path        string
	Operator    OperatorType
	IntValue    *int
	StringValue *string
	BoolValue   *bool
}

type OperatorType string

const (
	OpEquals      OperatorType = "eq"
	OpGreaterThan OperatorType = "gt"
	OpContains    OperatorType = "contains"
	OpNotContains OperatorType = "not_contains"
)

type HTTPRequestResponseVariable struct {
	Name string
	Path string
}

type CLIStepResult struct {
	CLICommandResult  *CLICommandResult
	HTTPRequestResult *HTTPRequestResult
}

type CLICommandResult struct {
	ExitCode     int
	FinalCommand string `json:"-"`
	Stdout       string
	Variables    map[string]string
}

type HTTPRequestResult struct {
	Err              string `json:"-"`
	StatusCode       int
	ResponseHeaders  map[string]string
	ResponseTrailers map[string]string
	BodyString       string
	Variables        map[string]string
	Request          CLIStepHTTPRequest
}

const BaseURLOverrideRequired = "override"

type Lesson struct {
	UUID                     string             `json:"UUID"`
	Title                    string             `json:"Title"`
	LessonDataMultipleChoice MultipleChoiceData `json:"LessonDataMultipleChoice"`
	LessonDataCodeTests      CodeData           `json:"LessonDataCodeTests"`
	LessonDataCLI            CLIData            `json:"LessonDataCLI"`
}

type Check struct {
	ContainsAll  []string `json:"ContainsAll"`
	MatchesOne   []string `json:"MatchesOne"`
	ContainsNone []string `json:"ContainsNone"`
}

type Response struct {
	Lesson struct {
		UUID             string `json:"UUID"`
		Slug             string `json:"Slug"`
		Type             string `json:"Type"`
		CourseUUID       string `json:"CourseUUID"`
		CourseTitle      string `json:"CourseTitle"`
		CourseSlug       string `json:"CourseSlug"`
		ChapterUUID      string `json:"ChapterUUID"`
		ChapterTitle     string `json:"ChapterTitle"`
		ChapterSlug      string `json:"ChapterSlug"`
		LessonDataManual struct {
			Readme string `json:"Readme"`
		}
		LessonDataTextInput struct {
			Readme        string `json:"Readme"`
			TextInputData Check  `json:"TextInputData"`
		}
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
		LessonDataCodeOutput struct {
			ProgLang           string
			StarterFiles       []StarterFile `json:"StarterFiles"`
			Readme             string        `json:"Readme"`
			CodeExpectedOutput string        `json:"CodeExpectedOutput"`
		} `json:"LessonDataCodeOutput"`
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
		LessonDataCLI struct {
			Readme  string
			CLIData `json:"CLIData"`
		} `json:"LessonDataCLI"`
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
	CHECK_URL           = BASE_API_URL + "/lessons/%s/checks"
	COURSE_URL          = BASE_API_URL + "static/courses/slug/"
	COURSE_PROGRESS_URL = BASE_API_URL + "course_progress_by_lesson/"
	TRACK_URL           = BASE_API_URL + "static/tracks/"
	TRACKS_URL          = BASE_API_URL + "static/tracks"
)

type errMsg struct {
	err error
}

type (
	stepStartMsg struct{ cmd string }
	stepDoneMsg  struct{ stdout string }
	CLIErr       struct {
		err    error
		stdout string
	}
	CLIDoneMsg struct{}
)

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

func (m *Model) getDirSuggestions() tea.Cmd {
	return func() tea.Msg {
		dir := m.dir.Value()

		if dir != "" {
			if entries, err := os.ReadDir(dir); err == nil {
				m.setDirSuggestions(dir, entries)
			} else if path.Dir(dir) != "" {
				if entries, err := os.ReadDir(dir); err == nil {
					m.setDirSuggestions(dir, entries)
				}
			}
		}
		return nil
	}
}

func (m *Model) setDirSuggestions(root string, entries []os.DirEntry) {
	s := funk.Map(entries, func(entry os.DirEntry) string {
		isHiddenFile := []rune(entry.Name())[0] == 46
		if entry.IsDir() && !isHiddenFile {
			return path.Join(root, entry.Name())
		}
		return ""
	})
	s = funk.Filter(s, func(a string) bool {
		return a != ""
	})
	switch s := s.(type) {
	case []string:
		m.dir.SetSuggestions(s)
	}
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
	dir                    textinput.Model
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
	dir := textinput.New()
	if d, err := os.Getwd(); err == nil {
		dir.SetValue(d)
	}

	dir.ShowSuggestions = true
	dir.Prompt = "Enter Directory: "
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
		dir:                    dir,
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
	m.dir, cmd = m.dir.Update(msg)
	cmds = append(cmds, cmd)
	cmds = append(cmds, m.getDirSuggestions())

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
		case "e":
			switch m.state {
			case CodeTestSuccess:
				cmds = append(cmds, m.openEditor())
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
					m.lessonURL = ""
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
					m.state = Git
					cmds = append(cmds, m.commitRepo())
				case QuestionFailed:
					m.state = Git
					cmds = append(cmds, m.commitRepo())
				case CodeTest:
					// TODO: add testing for lessons without unit tests
					cmds = append(cmds, m.testCode())
				case CodeTestFailed:
					cmds = append(cmds, m.openEditor())
				case CodeTestSuccess:
					cmds = append(cmds, m.commitRepo())
				case CheckOutput:
					cmds = append(cmds, m.CheckOutput())
				case OutputSuccess:
					cmds = append(cmds, m.commitRepo())
				case OutputFail:
					cmds = append(cmds, m.openEditor())
				case InputDir:
					m.dir.Blur()
					cmds = append(cmds, m.CLIChecks())
				case CLIDone:
					cmds = append(cmds, m.commitRepo())
				case CLIFailed:
					cmds = append(cmds, m.openEditor())
				case InputSuccess:
					cmds = append(cmds, m.commitRepo())
				case InputFail:
					cmds = append(cmds, m.openEditor())
				case Git:
					cmds = append(cmds, m.getNextLesson())
				case CourseFinished:
					m.state = TrackSelect
					if reflect.ValueOf(m.tracksResponse).IsZero() {
						cmds = append(cmds, func() tea.Msg { return request[TracksResponse](TRACKS_URL) })
					} else {
						m.state = TrackSelect
						m.list = m.createList(m.tracksResponse)
					}
				case Failed:
					cmds = append(cmds, tea.Quit)
				}
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.dir.Width = m.width / 2
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
		case CLICheck:
			cmds = append(cmds, m.CLIChecks())
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
		os.WriteFile(".last", []byte(m.lessonURL), 0o666)

		m.state = WriteFiles
		cmds = append(cmds, m.createCodeFiles())
	case stepStartMsg:
		m.content += fmt.Sprintf("󰣇 ❯ %s\n", msg.cmd)
		m.viewport = m.updateViewport()
	case stepDoneMsg:
		m.content += fmt.Sprintf("%s\n\n", msg.stdout)
		m.viewport = m.updateViewport()
	case CLIDoneMsg:
		m.content += "done"
		m.viewport = m.updateViewport()
		m.state = CLIDone
	case CLIErr:
		m.state = CLIFailed
		m.content = msg.err.Error() + msg.stdout
		m.viewport = m.updateViewport()
	case []*exec.Cmd:
		m.state = Git
		m.content = getCmdPipe(msg[0].Stdout)
		m.viewport = m.updateViewport()
		cmds = append(cmds, func() tea.Msg {
			return m.updateCmdOutput(msg)
		})
	case *exec.Cmd:
		m.state = NextLesson
		cmds = append(cmds, m.getNextLesson())
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
		return m.formatPager(correctStyle)
	case CodeTestFailed:
		m.title = "Code Test Failed"
		return m.formatPager(incorrectStyle)
	case CheckOutput:
		return "Checking Output"
	case OutputSuccess:
		m.title = "Output Matches"
		return m.formatPager(correctStyle)
	case OutputFail:
		m.title = "Output does not match"
		return m.formatPager(incorrectStyle)
	case InputDir:
		return m.dir.View()
	case CLICheck, CLIDone:
		m.title = "Running commands"
		return m.formatPager()
	case CLIFailed:
		m.title = "Command failed"
		return m.formatPager()
	case Git:
		m.title = "Pushing to repo"
		return m.formatPager()
	case InputSuccess:
		m.title = "Input Matches"
		return m.formatPager()
	case InputFail:
		m.title = "Input does not Match"
		return m.formatPager()
	case NextLesson:
		return "Press Enter to continue to next lesson. ctrl+c: quit"
	case CourseFinished:
		return "Course Finished 🎊. Enter to Select Track"
	default:
		return "\n"
	}
}

func (m Model) formatPager(styles ...lipgloss.Style) string {
	for _, style := range styles {
		m.viewport.Style = style
		break
	}
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

func (m *Model) getLessonType() tea.Cmd {
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
		case "type_cli":
			m.dir.SetValue(m.lessonPath())
			m.state = InputDir
			m.dir.Focus()
		case "type_text_input":
			m.state = CheckInput
			var err error
			m.response.Lesson.LessonDataTextInput.TextInputData, err = m.getLessonCheck()
			if err != nil {
				if err.Error() != "HTTP request failed with status: 403 Forbidden" {
					return errMsg{err: err}
				}
				m.state = Git
				return m.commitRepo()()
			} else {
				return m.CheckInput()()
			}
		case "type_manual":
			return m.commitRepo()()
		}
		return *m
	}
}

func (m Model) getLessonCheck() (Check, error) {
	t := request[Check](fmt.Sprintf(CHECK_URL, m.response.Lesson.UUID))
	switch t := t.(type) {
	case *Check:
		return *t, nil
	case errMsg:
		return Check{}, t.err
	default:
		return Check{}, errors.New("failed to request lesson check")
	}
}

func (m *Model) CheckInput() tea.Cmd {
	return func() tea.Msg {
		c, err := os.ReadFile(path.Join(m.lessonPath(), "input.txt"))
		content := string(c)
		if err != nil {
			return errMsg{
				err: fmt.Errorf("could not read user input file:\n %s", err),
			}
		}
		check := m.response.Lesson.LessonDataTextInput.TextInputData

		if check.ContainsAll != nil {
			var isError bool
			str := "Expect input file to contain all of:"
			for _, t := range check.ContainsAll {
				str += fmt.Sprintf("\n      - '%s'", t)
				if !strings.Contains(content, t) {
					isError = true
				}
			}
			if isError {
				str += fmt.Sprintf("\nContent is:\n%s", content)
				m.state = InputFail
				m.content = str
				m.viewport = m.updateViewport()
				return *m
			}
		}
		if check.MatchesOne != nil {
			isError := true
			str := "Expect input file to contain one of:"
			for _, t := range check.MatchesOne {
				str += fmt.Sprintf("\n      - '%s'", t)
				if strings.Contains(content, t) {
					isError = false
				}
			}
			if isError {
				str += fmt.Sprintf("\nContent is:\n%s", content)
				m.state = InputFail
				m.content = str
				m.viewport = m.updateViewport()
				return *m
			}
		}
		if check.ContainsNone != nil {
			isError := false
			str := "Expect input file to not contain any of:"
			for _, t := range check.ContainsNone {
				str += fmt.Sprintf("\n      - '%s'", t)
				if strings.Contains(content, t) {
					isError = true
				}
			}
			if isError {
				str += fmt.Sprintf("\nContent is:\n%s", content)
				m.state = InputFail
				m.content = str
				m.viewport = m.updateViewport()
				return *m
			}
		}
		m.content = content
		m.state = InputSuccess
		m.viewport = m.updateViewport()
		return *m
	}
}

func (m *Model) CLIChecks() tea.Cmd {
	return func() tea.Msg {
		cliData := m.response.Lesson.LessonDataCLI.CLIData
		variables := make(map[string]string)

		// prefer overrideBaseURL if provided, otherwise use BaseURLDefault

		for _, step := range cliData.Steps {
			if step.CLICommand != nil {
				p.Send(stepStartMsg{cmd: step.CLICommand.Command})
				result := m.runCLICommand(*step.CLICommand, variables)
				for _, test := range step.CLICommand.Tests {
					if err := m.isCLIError(result, &test, result.Variables); err != nil {
						return CLIErr{err: err, stdout: result.Stdout}
					}
				}
				p.Send(stepDoneMsg{stdout: result.Stdout})
			} else if step.HTTPRequest != nil {
				return errMsg{errors.New("unimplemented step: HTTPRequest")}
			} else {
				return errMsg{errors.New("unable to run lesson: missing step")}
			}
		}
		os.WriteFile(path.Join(m.lessonPath(), ".dir"), []byte(m.dir.Value()), 0o655)
		return CLIDoneMsg{}
	}
}

func (m Model) isCLIError(result CLICommandResult, test *CLICommandTest, variables map[string]string) error {
	if test.ExitCode != nil && *test.ExitCode != result.ExitCode {
		return fmt.Errorf("expect exit code %d\n", *test.ExitCode)
	}
	if test.StdoutLinesGt != nil && *test.StdoutLinesGt > len(strings.Split(result.Stdout, "\n")) {
		return fmt.Errorf("expect > %d lines on stdout", *test.StdoutLinesGt)
	}
	if test.StdoutContainsAll != nil {
		str := "Expect stdout to contain all of:"
		var hasError bool
		for _, contains := range test.StdoutContainsAll {
			interpolatedContains := InterpolateVariables(contains, variables)
			str += fmt.Sprintf("\n      - '%s'", interpolatedContains)
			if !strings.Contains(result.Stdout, interpolatedContains) {
				hasError = true
			}
		}
		if hasError {
			return errors.New(str)
		}
	}
	if test.StdoutContainsNone != nil {
		str := "Expect stdout to contain none of:"
		var hasError bool
		for _, containsNone := range test.StdoutContainsNone {
			interpolatedContains := InterpolateVariables(containsNone, variables)
			str += fmt.Sprintf("\n      - '%s'", interpolatedContains)
			if strings.Contains(result.Stdout, interpolatedContains) {
				hasError = true
			}
		}
		if hasError {
			return errors.New(str)
		}
	}
	return nil
}

func (m Model) runCLICommand(command CLIStepCLICommand, variables map[string]string) (result CLICommandResult) {
	finalCommand := InterpolateVariables(command.Command, variables)
	result.FinalCommand = finalCommand

	cmd := exec.Command("sh", "-c", finalCommand)
	cmd.Dir = m.dir.Value()
	cmd.Env = append(os.Environ(), "LANG=en_US.UTF-8")
	b, err := cmd.CombinedOutput()
	result.Stdout = strings.TrimRight(string(b), " \n\t\r")
	if ee, ok := err.(*exec.ExitError); ok {
		result.ExitCode = ee.ExitCode()
	} else if err != nil {
		result.ExitCode = -2
		result.Stdout = err.Error()
	}
	result.Variables = variables
	return result
}

func InterpolateVariables(template string, vars map[string]string) string {
	r := regexp.MustCompile(`\$\{([^}]+)\}`)
	return r.ReplaceAllStringFunc(template, func(m string) string {
		// Extract the key from the match, which is in the form ${key}
		key := strings.TrimSuffix(strings.TrimPrefix(m, "${"), "}")
		if val, ok := vars[key]; ok {
			return val
		}
		return m // return the original placeholder if no substitution found
	})
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
	if reflect.ValueOf(m.response.Lesson.LessonDataCLI).IsZero() {
		return path.Join(m.response.Lesson.CourseSlug, m.response.Lesson.ChapterSlug, m.response.Lesson.Slug)
	}
	return m.response.Course.Slug
}

func (m Model) createCodeFiles() tea.Cmd {
	return func() tea.Msg {
		fmt.Printf(
			"📂 Course %s, Chapter %s, Lesson %s\n", m.response.Lesson.CourseSlug,
			m.response.Lesson.ChapterSlug, m.response.Lesson.Slug,
		)

		// Create chapter directory if it doesn't exist
		exerciseDir := m.lessonPath()
		if err := os.MkdirAll(exerciseDir, 0o755); err != nil {
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
			if m.response.Lesson.LessonDataCodeCompletion.Readme != "" {
				starterFiles = m.response.Lesson.LessonDataCodeCompletion.StarterFiles
				readme = m.response.Lesson.LessonDataCodeCompletion.Readme
			} else {
				starterFiles = m.response.Lesson.LessonDataCodeOutput.StarterFiles
				readme = m.response.Lesson.LessonDataCodeOutput.Readme
			}
		case "type_choice":
			starterFiles = []StarterFile{}
			readme = m.response.Lesson.LessonDataMultipleChoice.Readme
			question := m.response.Lesson.LessonDataMultipleChoice.Question.Question
			choices := m.response.Lesson.LessonDataMultipleChoice.Question.Answers
			readme = fmt.Sprintf("%s\n# Question\n### %s\n- %s", readme, question, strings.Join(choices, "\n- "))
		case "type_cli":
			starterFiles = []StarterFile{}
			readme = m.response.Lesson.LessonDataCLI.Readme
		case "type_manual":
			starterFiles = []StarterFile{}
			readme = m.response.Lesson.LessonDataManual.Readme
		case "type_text_input":
			starterFiles = []StarterFile{{Name: "input.txt"}}
			readme = m.response.Lesson.LessonDataTextInput.Readme

		default:
			return errMsg{err: fmt.Errorf("unknown lesson type: %s", m.response.Lesson.Type)}
		}

		m.starterFiles = append(m.starterFiles, "README.md")
		for _, file := range starterFiles {
			if file.IsHidden {
				continue // Skip hidden files
			}
			filePath := filepath.Join(exerciseDir, file.Name)
			if _, err := os.Stat(filePath); err != nil {
				if err := os.WriteFile(filePath, []byte(file.Content), 0o644); err != nil {
					return errMsg{err: fmt.Errorf("failed to create %s: %v", filePath, err)}
				}
			}
			m.starterFiles = append(m.starterFiles, file.Name)
		}

		// Create README.md in the exercise directory
		readmePath := filepath.Join(exerciseDir, "README.md")
		if err := os.WriteFile(readmePath, []byte(readme), 0o644); err != nil {
			return errMsg{err: fmt.Errorf("failed to create README.md: %v", err)}
		}

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
	args := m.starterFiles[1:]
	if codeEditor == "" {
		codeEditor = "nvr"
		args = append(args, "-cc", fmt.Sprintf("terminal glow -p %s", m.starterFiles[0]), "-cc", "vsplit", "--remote-wait-silent")
	}

	cmd := exec.Command(codeEditor, args...)
	cmd.Dir = m.lessonPath()
	if m.response.Lesson.Type == "type_cli" || m.response.Lesson.Type == "type_manual" || m.response.Lesson.Type == "type_text_input" {
		cmd.Args = append(cmd.Args, "-cc", "lua vim.g.bootdev=true")
	}

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
			if m.response.Lesson.LessonDataCodeCompletion.ProgLang != "" {
				makeFile = ".lib/" + m.response.Lesson.LessonDataCodeCompletion.ProgLang + "/Makefile"
			} else {
				makeFile = ".lib/" + m.response.Lesson.LessonDataCodeOutput.ProgLang + "/Makefile"
			}
		}
		if _, err := os.Stat(makeFile); err != nil {
			return errMsg{
				err: fmt.Errorf("could not open MakeFile: %v", err),
			}
		}
		cmd := exec.Command("make", "-f", makeFile, m.lessonPath())

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
		var script string
		if m.response.Lesson.LessonDataCodeCompletion.Readme != "" {
			script = ".lib/" + m.response.Lesson.LessonDataCodeCompletion.ProgLang + "/run"
		} else {
			script = ".lib/" + m.response.Lesson.LessonDataCodeOutput.ProgLang + "/run"
		}
		if _, err := os.Stat(script); err != nil {
			return errMsg{
				err: fmt.Errorf("script to run program does not exist: %v", err),
			}
		}
		cmd := exec.Command("bash", script, m.lessonPath())

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

		var expectedOutput string
		if m.response.Lesson.LessonDataCodeCompletion.CodeExpectedOutput != "" {
			expectedOutput = m.response.Lesson.LessonDataCodeCompletion.CodeExpectedOutput
		} else {
			expectedOutput = m.response.Lesson.LessonDataCodeOutput.CodeExpectedOutput
		}

		if out.String() == expectedOutput {
			m.content = out.String()
			m.state = OutputSuccess
		} else {
			m.content = diff.CharacterDiff(out.String(), expectedOutput)
			m.state = OutputFail
		}
		m.viewport = m.updateViewport()
		return m
	}
}

func (m Model) commitRepo() tea.Cmd {
	return func() tea.Msg {
		var stdout bytes.Buffer
		if _, err := os.Stat(".git"); err != nil {
			return errMsg{
				err: fmt.Errorf("initialize git repo"),
			}
		}

		path := m.lessonPath()
		var cmds []*exec.Cmd
		cmds = append(cmds, exec.Command("git", "add", path))
		cmds[0].Stdout = &stdout
		cmds[0].Stderr = &bytes.Buffer{}

		courseSlug := m.response.Lesson.CourseSlug
		lessonNum := m.getLessonNumber()
		chapNum := m.getChapterNumber()
		msg := fmt.Sprintf("%s - Chapter %d - Lesson %d", courseSlug, chapNum, lessonNum)
		cmds = append(cmds, exec.Command("git", "commit", "-m", msg))
		args := []string{"push"}
		if pid, err := GetTracerPid(); err == nil && pid > 0 {
			args = append(args, "-n", "-v")
		}
		cmds = append(cmds, exec.Command("git", args...))

		m.state = Git
		return cmds
	}
}

func (m Model) updateCmdOutput(cmds []*exec.Cmd) tea.Msg {
	cmd := cmds[0]
	if cmd.Process == nil {
		fmt.Fprintf(cmd.Stdout, "󰣇 ❯ %v", strings.Join(cmd.Args, " "))
		if err := cmd.Start(); err != nil {
			return errMsg{
				err: fmt.Errorf("failed to start %v\n%v", strings.Join(cmd.Args, " "), err),
			}
		}
		cmd.Stderr = &bytes.Buffer{}
	} else {
		var status syscall.WaitStatus
		pid, err := syscall.Wait4(cmd.Process.Pid, &status, syscall.WNOHANG, nil)
		if err != nil {
			return errMsg{
				err: fmt.Errorf("failed to reap process %d: %v", cmd.Process.Pid, err),
			}
		}
		if pid > 0 {
			if status.Exited() {
				if status.ExitStatus() != 0 {
					fmt.Fprintf(cmd.Stdout, "\"%v\" failed with exit code %v\n%v", strings.Join(cmd.Args, " "), status.ExitStatus(), getCmdPipe(cmd.Stderr))
				}
				if len(cmds) == 1 {
					return cmd
				} else {
					cmds[1].Stdout = cmd.Stdout
					cmds = cmds[1:]
				}
			}
		}
	}
	return cmds
}

func getCmdPipe(w io.Writer) string {
	e := ""
	switch a := w.(type) {
	case *bytes.Buffer:
		e = a.String()
	}
	return e
}

func (m Model) getNextLesson() tea.Cmd {
	return func() tea.Msg {
		m.attempts = 0
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

func GetTracerPid() (int, error) {
	file, err := os.Open("/proc/self/status")
	if err != nil {
		return -1, fmt.Errorf("can't open process status file: %w", err)
	}
	defer file.Close()

	for {
		var tpid int
		num, err := fmt.Fscanf(file, "TracerPid: %d\n", &tpid)
		if err == io.EOF {
			break
		}
		if num != 0 {
			return tpid, nil
		}
	}

	return -1, errors.New("unknown format of process status file")
}
