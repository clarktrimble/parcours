package board

import (
	"image"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/pkg/errors"
)

const (
	gutter       = 1 // space between columns
	separator    = "-"
	headerHeight = 2  // header row + separator line
	jumpBy       = 10 // ranks to scroll with alt+up/down

	// Scroll acceleration
	accelWindow = 200 * time.Millisecond // reset if no repeat within this window
	accelMax    = 10                     // max step size
)

var (
	tableBorderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	hlRowStyle       = lipgloss.NewStyle().Background(lipgloss.Color("235"))
	hlCellStyle      = lipgloss.NewStyle().Background(lipgloss.Color("237"))
)

// Placeholder is a tea.Model that displays while waiting for data
type Placeholder struct {
	message string
}

func NewPlaceholder(message string) Placeholder {
	return Placeholder{message: message}
}

func (p Placeholder) Init() tea.Cmd                       { return nil }
func (p Placeholder) Update(tea.Msg) (tea.Model, tea.Cmd) { return p, nil }

func (p Placeholder) View() tea.View {
	return tea.NewView(p.message)
}

// File represents a file down the board.
// Has ambitions to carry type and/or formatter for its pieces.
type File struct {
	Name   string
	Width  int
	SrcIdx int // Index into source data
}

// Files is a slice of File with comparison support.
type Files []File

// Equal returns true if both slices have the same files.
func (f Files) Equal(other Files) (equal bool) {
	if len(f) != len(other) {
		return
	}
	for i := range f {
		if f[i].Name != other[i].Name || f[i].Width != other[i].Width {
			return
		}
	}
	equal = true
	return
}

// PieceMsg is the interface for messages from interactive pieces.
type PieceMsg interface {
	IsPieceMsg()
}

// Square represents a square of the board.
type Square struct {
	piece tea.Model
}

// Rank represents a rank across the board.
type Rank struct {
	squares []Square
}

// NewRank creates a Rank from a slice of pieces.
func NewRank(pieces []tea.Model) Rank {
	squares := make([]Square, len(pieces))
	for i, piece := range pieces {
		squares[i] = Square{
			piece: piece,
		}
	}
	return Rank{squares: squares}
}

// Append adds a piece to the rank.
func (r *Rank) Append(piece tea.Model) {
	r.squares = append(r.squares, Square{piece: piece})
}

// Ranks is a slice of Rank.
type Ranks []Rank

// Board represents a grid of squares with ranks and files.
type Board struct {
	ranks    []Rank
	files    []File
	position position
	width    int // Number of files
	height   int // Number of ranks

	// Viewport (height from length of ranks)
	vpw         int
	initialized bool
	fileOffset  int // Index of leftmost visible file (for horizontal scrolling)

	// Layout cache - updated when fileOffset or viewport changes
	rankAreas []image.Rectangle
	fileEnd   int // Index (exclusive) of last file to draw
	fullSpan  int

	// Edit mode - when true, keys go to focused piece instead of nav
	editMode bool

	// Scroll acceleration
	repeatDir   string
	repeatTime  time.Time
	repeatCount int
}

// New creates a Board with viewport size.
func New(ranks []Rank, files []File, rank, file, vpw int) (Board, error) {
	brd := Board{
		ranks:       ranks,
		files:       files,
		width:       len(files),
		height:      len(ranks),
		position:    position{file: file, rank: rank},
		vpw:         vpw,
		initialized: true,
	}

	err := brd.validate()
	if err != nil {
		return brd, err
	}

	brd.setPositions()
	brd = brd.updateLayout()
	return brd, nil
}

func (brd Board) Init() tea.Cmd {
	return nil
}

func (brd Board) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	switch msg := msg.(type) {

	case SizeMsg:
		brd.vpw = msg.Width
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
			brd.position.rank = 0
		case Bottom:
			brd.position.rank = brd.height - 1
		}
		return brd, brd.positionCmd()

	case tea.KeyPressMsg:
		if brd.editMode {
			// Edit mode: Enter exits, everything else goes to piece
			if msg.String() == "enter" {
				brd.editMode = false
				return brd, nil
			}
			return brd.updatePiece(msg)
		}

		// Nav mode
		switch msg.String() {

		case "i":
			// Todo: editMode only makes sense for textinput
			brd.editMode = true
			return brd, nil

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
				return NavMsg{Direction: NavUp, Count: brd.height}
			}

		case "pgdown":
			return brd, func() tea.Msg {
				return NavMsg{Direction: NavDown, Count: brd.height}
			}

		case "alt+up":
			return brd.move(NavUp, jumpBy)

		case "alt+down":
			return brd.move(NavDown, jumpBy)

		case "g", "alt+pgup":
			brd.position.rank = 0
			return brd, tea.Batch(brd.positionCmd(), navCmd(NavTop))

		case "G", "alt+pgdown":
			brd.position.rank = brd.height - 1
			return brd, tea.Batch(brd.positionCmd(), navCmd(NavBottom))
		}
	}

	return brd.updatePiece(msg)
}

// updatePiece passes a message to the focused piece
func (brd Board) updatePiece(msg tea.Msg) (Board, tea.Cmd) {

	r := brd.position.rank
	f := brd.position.file

	square := brd.ranks[r].squares[f]
	piece, cmd := square.piece.Update(msg)
	brd.ranks[r].squares[f].piece = piece

	return brd, cmd
}

func (brd Board) View() tea.View {
	if !brd.initialized {
		return tea.NewView("loading...")
	}

	canvas := lipgloss.NewCanvas(brd.vpw, headerHeight+len(brd.ranks))
	brd.draw(canvas)
	brd.drawHighlight(canvas)

	return tea.NewView(canvas)
}

// unexported

type position struct {
	rank int
	file int
}

// areasIter yields drawing areas at successive vertical offsets
type areasIter struct {
	template []image.Rectangle
	areas    []image.Rectangle
	offset   int
}

func newAreasIter(template []image.Rectangle) *areasIter {
	return &areasIter{template: template, areas: make([]image.Rectangle, len(template)), offset: 0}
}

func (ai *areasIter) next() []image.Rectangle {
	for i, t := range ai.template {
		ai.areas[i] = image.Rect(t.Min.X, ai.offset, t.Max.X, ai.offset+1)
	}
	ai.offset++
	return ai.areas
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

// replace replaces a board's ranks.
func (brd Board) replace(ranks []Rank) (board Board, err error) {

	brd.ranks = ranks
	brd.height = len(ranks)

	// Clamp cursor if new height is smaller
	if brd.position.rank >= brd.height {
		brd.position.rank = max(0, brd.height-1)
	}

	err = brd.validate()
	if err != nil {
		return
	}

	brd.setPositions()
	board = brd
	return
}

// applyBgToArea sets the background color on all cells in the given area
func applyBgToArea(canvas *lipgloss.Canvas, x, y, width int, sty lipgloss.Style) {

	bg := sty.GetBackground()
	for cx := x; cx < x+width; cx++ {

		cell := canvas.CellAt(cx, y)
		if cell != nil {
			cell.Style.Bg = bg
			canvas.SetCell(cx, y, cell)
		}
	}
}

func (brd Board) updateLayout() Board {
	// Todo: look at naming and comment properly
	// Clear previous layout
	brd.rankAreas = brd.rankAreas[:0]

	x := 0
	for i := brd.fileOffset; i < len(brd.files); i++ {
		// Stop when viewport is full
		if x >= brd.vpw {
			break
		}
		// Add column rect at header row (Y=0)
		w := brd.files[i].Width
		brd.rankAreas = append(brd.rankAreas, image.Rect(x, 0, x+w, 1))
		x += w + gutter
		brd.fileEnd = i + 1
	}
	brd.fullSpan = x - gutter
	return brd
}

func (brd Board) drawHighlight(canvas *lipgloss.Canvas) {
	selectedY := headerHeight + brd.position.rank
	visualFile := brd.position.file - brd.fileOffset
	cellArea := brd.rankAreas[visualFile]

	applyBgToArea(canvas, 0, selectedY, brd.fullSpan, hlRowStyle)
	applyBgToArea(canvas, cellArea.Min.X, selectedY, cellArea.Dx(), hlCellStyle)
}

func (brd Board) draw(canvas *lipgloss.Canvas) {
	areas := newAreasIter(brd.rankAreas)

	// Headers
	hdrs := areas.next()
	for i, file := range brd.files[brd.fileOffset:brd.fileEnd] {
		uv.NewStyledString(file.Name).Draw(canvas, hdrs[i])
	}

	// Separator
	areas.next()
	sepArea := image.Rect(0, areas.offset-1, brd.fullSpan, areas.offset)
	str := strings.Repeat(separator, brd.fullSpan)
	uv.NewStyledString(tableBorderStyle.Render(str)).Draw(canvas, sepArea)

	// Ranks
	for _, rank := range brd.ranks {
		rects := areas.next()
		for i, square := range rank.squares[brd.fileOffset:brd.fileEnd] {
			square.piece.View().Content.Draw(canvas, rects[i])
		}
	}
}

// positionCmd returns a command that sends the current cursor position
func (brd Board) positionCmd() tea.Cmd {
	pos := SquareMsg{
		Rank: brd.position.rank,
		File: brd.position.file,
	}
	return func() tea.Msg { return pos }
}

func navCmd(dir string) tea.Cmd {
	return func() tea.Msg { return NavMsg{Direction: dir, Count: 1} }
}

// accelMove moves with acceleration - step size increases with rapid repeats
func (brd Board) accelMove(dir string) (Board, tea.Cmd) {
	now := time.Now()
	if brd.repeatDir == dir && now.Sub(brd.repeatTime) < accelWindow {
		brd.repeatCount++
	} else {
		brd.repeatCount = 1
		brd.repeatDir = dir
	}
	brd.repeatTime = now
	step := min(brd.repeatCount, accelMax)
	return brd.move(dir, step)
}

// move moves the cursor by n in the given direction.
// If the move exceeds the board boundary, cursor stops at the edge
// and a NavMsg is emitted with the remaining count for the parent to handle.
// Note: positionCmd is always emitted at boundary even if cursor didn't move.
func (brd Board) move(dir string, n int) (Board, tea.Cmd) {
	switch dir {
	case NavUp:
		if brd.position.rank >= n {
			brd.position.rank -= n
			return brd, brd.positionCmd()
		}
		// Hit top boundary
		remainder := n - brd.position.rank
		brd.position.rank = 0
		return brd, tea.Batch(
			brd.positionCmd(),
			func() tea.Msg { return NavMsg{Direction: NavUp, Count: remainder} },
		)

	case NavDown:
		maxRank := brd.height - 1
		if brd.position.rank+n <= maxRank {
			brd.position.rank += n
			return brd, brd.positionCmd()
		}
		// Hit bottom boundary
		remainder := n - (maxRank - brd.position.rank)
		brd.position.rank = maxRank
		return brd, tea.Batch(
			brd.positionCmd(),
			func() tea.Msg { return NavMsg{Direction: NavDown, Count: remainder} },
		)

	// Horizontal movement ignores n for now
	case NavLeft:
		if brd.position.file == 0 {
			return brd, nil
		}
		brd.position.file--
		brd = brd.checkFileOffset()

	case NavRight:
		if brd.position.file == brd.width-1 {
			return brd, nil
		}
		brd.position.file++
		brd = brd.checkFileOffset()
	}

	return brd, brd.positionCmd()
}

func (brd Board) checkFileOffset() Board {
	switch {
	case brd.position.file < brd.fileOffset:
		brd.fileOffset = brd.position.file
	case brd.position.file >= brd.fileEnd:
		brd.fileOffset = brd.offsetFor(brd.position.file)
	default:
		return brd
	}
	return brd.updateLayout()
}

// offsetFor calculates the offset which will lead to a given file being shown at the right edge of board
func (brd Board) offsetFor(file int) int {
	var width int
	for i := file; i >= 0; i-- {
		width += brd.files[i].Width + gutter
		if width-gutter > brd.vpw {
			return i + 1
		}
	}
	return 0
}

func (brd Board) validate() error {

	if len(brd.ranks) == 0 || len(brd.files) == 0 {
		return errors.Errorf("board requires non-zero ranks and files")
	}

	if len(brd.ranks) != brd.height {
		return errors.Errorf("ranks length %d does not match height %d", len(brd.ranks), brd.height)
	}
	if len(brd.files) != brd.width {
		return errors.Errorf("files length %d does not match width %d", len(brd.files), brd.width)
	}

	for i, r := range brd.ranks {
		if len(r.squares) != brd.width {
			return errors.Errorf("rank %d length does not equal width", i)
		}
	}

	if brd.position.rank < 0 || brd.position.rank >= brd.height {
		return errors.Errorf("rank %d out of bounds [0, %d)", brd.position.rank, brd.height)
	}
	if brd.position.file < 0 || brd.position.file >= brd.width {
		return errors.Errorf("file %d out of bounds [0, %d)", brd.position.file, brd.width)
	}

	return nil
}
