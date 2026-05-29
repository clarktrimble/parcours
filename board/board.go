package board

import (
	"image"
	"math"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/pkg/errors"

	"parcours/message"
)

const (
	gutter       = 1 // space between columns
	separator    = "─"
	headerHeight = 2  // header row + separator line
	jumpBy       = 10 // ranks to scroll with alt+up/down

	accelWindow = 200 * time.Millisecond // reset if no repeat within this window
	accelMax    = 10                     // max step size
	accelCurve  = 0.5                    // log ramp shape: lower = more gradual
)

var (
	tableBorderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	hlRowStyle       = lipgloss.NewStyle().Background(lipgloss.Color("235"))
	hlCellStyle      = lipgloss.NewStyle().Background(lipgloss.Color("237"))
)

// Board represents a grid of squares with ranks and files.
type Board struct {
	ranks  []Rank
	files  []File
	cursor position
	keyMap KeyMap

	initialized bool              // set after first SizeMsg
	vpWidth     int               // viewpoint width (height is header + ranks)
	offset      int               // leftmost visible file index
	end         int               // rightmost visible file index (exclusive)
	layout      []image.Rectangle // rectangles for rank and file layout

	editMode bool // send all keys to focused piece

	accelDir  string    // direction of acceleration
	accelAt   time.Time // time of last accelerated move
	accelStep int       // current step size
}

// New creates a Board with viewport size.
func New(ranks []Rank, files []File, rank, file, vpw int) (Board, error) {

	layout, end := computeLayout(files, 0, vpw)

	brd := Board{
		ranks:       ranks,
		files:       files,
		cursor:      position{file: file, rank: rank},
		keyMap:      DefaultKeyMap(),
		vpWidth:     vpw,
		initialized: true,
		layout:      layout,
		end:         end,
	}

	err := brd.validate()
	if err != nil {
		return brd, err
	}

	brd.setPositions()
	return brd, nil
}

func (brd Board) Init() tea.Cmd {
	return nil
}

func (brd Board) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		brd.vpWidth = msg.Width
		brd.initialized = true
		return brd.checkFileOffset(), nil

	case ReplaceMsg:
		newBrd, err := brd.replace(msg.Ranks)
		if err != nil {
			return brd, func() tea.Msg { return err }
		}
		return newBrd, newBrd.positionCmd()

	case MoveToMsg:
		switch msg.MoveTo {
		case Top:
			brd.cursor.rank = 0
		case Bottom:
			brd.cursor.rank = len(brd.ranks) - 1
		}
		return brd, brd.positionCmd()

	case message.ReportHintsMsg:
		return brd, brd.hintsCmd()

	case tea.KeyPressMsg:
		if brd.editMode {
			// Edit mode: Enter exits, everything else goes to piece
			if msg.String() == "enter" {
				brd.editMode = false
				return brd, hintsChangedCmd()
			}
			return brd.updatePiece(msg)
		}

		// Nav mode
		switch msg.String() {

		case "i":
			// Todo: editMode only makes sense for textinput
			brd.editMode = true
			return brd, hintsChangedCmd()

		case "up", "k":
			return brd.accelMove(NavUp)
		case "down", "j":
			return brd.accelMove(NavDown)
		case "left", "h":
			return brd.move(NavLeft, 1)
		case "right", "l":
			return brd.move(NavRight, 1)

		case "pgup":
			return brd, func() tea.Msg {
				return NavMsg{Direction: NavUp, Count: len(brd.ranks)}
			}

		case "pgdown":
			return brd, func() tea.Msg {
				return NavMsg{Direction: NavDown, Count: len(brd.ranks)}
			}

		case "alt+up":
			return brd.move(NavUp, jumpBy)

		case "alt+down":
			return brd.move(NavDown, jumpBy)

		case "g", "alt+pgup":
			brd.cursor.rank = 0
			return brd, tea.Batch(brd.positionCmd(), navCmd(NavTop))

		case "G", "alt+pgdown":
			brd.cursor.rank = len(brd.ranks) - 1
			return brd, tea.Batch(brd.positionCmd(), navCmd(NavBottom))
		}
	}

	return brd.updatePiece(msg)
}

func (brd Board) View() tea.View {
	if !brd.initialized {
		return tea.NewView("loading...")
	}

	canvas := lipgloss.NewCanvas(brd.vpWidth, headerHeight+len(brd.ranks))
	brd.draw(canvas)
	brd.highlight(canvas)

	return tea.NewView(canvas)
}

// unexported

type position struct {
	rank int
	file int
}

func (brd *Board) setPositions() {
	for r, rank := range brd.ranks {
		for f := range rank.squares {
			piece := rank.squares[f].piece
			updated, _ := piece.Update(PositionMsg{Rank: r, File: f})
			brd.ranks[r].squares[f].piece = updated
		}
	}
}

func (brd Board) replace(ranks []Rank) (Board, error) {
	brd.ranks = ranks

	// clamp cursor if new height is smaller
	if brd.cursor.rank >= len(brd.ranks) {
		brd.cursor.rank = max(0, len(brd.ranks)-1)
	}

	err := brd.validate()
	if err != nil {
		return brd, err
	}

	brd.setPositions()
	return brd, nil
}

func (brd Board) updatePiece(msg tea.Msg) (Board, tea.Cmd) {

	r := brd.cursor.rank
	f := brd.cursor.file

	square := brd.ranks[r].squares[f]
	piece, cmd := square.piece.Update(msg)
	brd.ranks[r].squares[f].piece = piece

	return brd, cmd
}

func setBackground(canvas *lipgloss.Canvas, x, y, width int, sty lipgloss.Style) {

	bg := sty.GetBackground()
	for cx := x; cx < x+width; cx++ {

		cell := canvas.CellAt(cx, y)
		if cell != nil {
			cell.Style.Bg = bg
			canvas.SetCell(cx, y, cell)
		}
	}
}

func computeLayout(files []File, offset, vpWidth int) (layout []image.Rectangle, end int) {

	// track horizontal position across the veiwport
	var x int

	// range thru files from offset until file is off the viewport
	for i := offset; i < len(files); i++ {
		if x >= vpWidth {
			break
		}

		// add a layout rectangle for this file's width
		fileWidth := files[i].Width
		layout = append(layout, image.Rect(x, 0, x+fileWidth, 1))

		// dont foreget gutter when bumping x and track index of last visible
		x += fileWidth + gutter
		end = i + 1
	}

	return
}

func (brd Board) contentWidth() int {
	if len(brd.layout) == 0 {
		return 0
	}
	return brd.layout[len(brd.layout)-1].Max.X
}

func (brd Board) highlight(canvas *lipgloss.Canvas) {

	highlightY := headerHeight + brd.cursor.rank
	setBackground(canvas, 0, highlightY, brd.contentWidth(), hlRowStyle)

	cellArea := brd.layout[brd.cursor.file-brd.offset]
	setBackground(canvas, cellArea.Min.X, highlightY, cellArea.Dx(), hlCellStyle)
}

func (brd Board) draw(canvas *lipgloss.Canvas) {
	areas := newRectIter(brd.layout)

	// Headers
	hdrs := areas.next()
	for i, file := range brd.files[brd.offset:brd.end] {
		uv.NewStyledString(file.Header).Draw(canvas, hdrs[i])
	}

	// Separator
	areas.next()
	sepArea := image.Rect(0, areas.offset-1, brd.contentWidth(), areas.offset)
	str := strings.Repeat(separator, brd.contentWidth())
	uv.NewStyledString(tableBorderStyle.Render(str)).Draw(canvas, sepArea)

	// Ranks
	for _, rank := range brd.ranks {
		rects := areas.next()
		for i, square := range rank.squares[brd.offset:brd.end] {
			square.piece.View().Content.Draw(canvas, rects[i])
		}
	}
}

// accelMove moves with acceleration - step size increases with rapid repeats
func (brd Board) accelMove(dir string) (Board, tea.Cmd) {
	now := time.Now()
	if brd.accelDir == dir && now.Sub(brd.accelAt) < accelWindow {
		brd.accelStep++
	} else {
		brd.accelStep = 1
		brd.accelDir = dir
	}
	brd.accelAt = now

	// Logarithmic: slower ramp, smoother feel
	step := 1 + int(accelCurve*math.Log2(float64(brd.accelStep+1)))

	// Linear: constant ramp
	// step := brd.accelStep

	return brd.move(dir, min(step, accelMax))
}

// move moves the cursor by n in the given direction.
// If the move exceeds the board boundary, cursor stops at the edge
// and a NavMsg is emitted with the remaining count for the parent to handle.
// Note: positionCmd is always emitted at boundary even if cursor didn't move.
func (brd Board) move(dir string, n int) (Board, tea.Cmd) {
	switch dir {
	case NavUp:
		if brd.cursor.rank >= n {
			brd.cursor.rank -= n
			return brd, brd.positionCmd()
		}
		// Hit top boundary
		remainder := n - brd.cursor.rank
		brd.cursor.rank = 0
		return brd, tea.Batch(
			brd.positionCmd(),
			func() tea.Msg { return NavMsg{Direction: NavUp, Count: remainder} },
		)

	case NavDown:
		maxRank := len(brd.ranks) - 1
		if brd.cursor.rank+n <= maxRank {
			brd.cursor.rank += n
			return brd, brd.positionCmd()
		}
		// Hit bottom boundary
		remainder := n - (maxRank - brd.cursor.rank)
		brd.cursor.rank = maxRank
		return brd, tea.Batch(
			brd.positionCmd(),
			func() tea.Msg { return NavMsg{Direction: NavDown, Count: remainder} },
		)

	// Horizontal movement ignores n for now
	case NavLeft:
		if brd.cursor.file == 0 {
			return brd, nil
		}
		brd.cursor.file--
		brd = brd.checkFileOffset()

	case NavRight:
		if brd.cursor.file == len(brd.files)-1 {
			return brd, nil
		}
		brd.cursor.file++
		brd = brd.checkFileOffset()
	}

	return brd, brd.positionCmd()
}

func (brd Board) checkFileOffset() Board {
	switch {
	case brd.cursor.file < brd.offset:
		// scrolling to the left
		brd.offset = brd.cursor.file
	case brd.cursor.file >= brd.end:
		// scrolling to the right
		brd.offset = brd.offsetFor(brd.cursor.file)
	default:
		// not scrolling view horizontally
		return brd
	}
	brd.layout, brd.end = computeLayout(brd.files, brd.offset, brd.vpWidth)
	return brd
}

// offsetFor calculates the offset which will show a given file at the right edge of board
func (brd Board) offsetFor(file int) int {
	var width int
	for i := file; i >= 0; i-- {
		width += brd.files[i].Width + gutter
		if width-gutter > brd.vpWidth {
			return i + 1
		}
	}
	return 0
}

func (brd Board) validate() error {

	// Todo: maybe guard as needed instead?
	if len(brd.ranks) == 0 || len(brd.files) == 0 {
		return errors.Errorf("board requires non-zero ranks and files")
	}

	for i, r := range brd.ranks {
		if len(r.squares) != len(brd.files) {
			return errors.Errorf("rank %d has %d squares, expected %d", i, len(r.squares), len(brd.files))
		}
	}

	if brd.cursor.rank < 0 || brd.cursor.rank >= len(brd.ranks) {
		return errors.Errorf("rank %d out of bounds [0, %d)", brd.cursor.rank, len(brd.ranks))
	}
	if brd.cursor.file < 0 || brd.cursor.file >= len(brd.files) {
		return errors.Errorf("file %d out of bounds [0, %d)", brd.cursor.file, len(brd.files))
	}

	return nil
}

func (brd Board) hintsCmd() tea.Cmd {
	hints := brd.keyMap.Hints(brd.editMode)
	return func() tea.Msg {
		return message.HintsMsg{Hints: hints}
	}
}

func hintsChangedCmd() tea.Cmd {
	return func() tea.Msg {
		return message.HintsChangedMsg{}
	}
}
