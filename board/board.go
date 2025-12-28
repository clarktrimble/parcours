package board

import (
	"fmt"
	"image"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/pkg/errors"

	"parcours/message"
	"parcours/style"
)

const (
	gutter = 1 // space between columns
)

type File interface {
	Name() string
	Width() int
}

// Piece represents a board piece that can update and render itself.
type Piece interface {
	Update(tea.Msg) (Piece, tea.Cmd)
	View() tea.View
	Render() string // Deprecated: use View() instead
	Value() string  // Returns the raw value (for filtering, etc.) Todo: nt.Value ??
}

// PieceMsg is the interface for messages from interactive pieces.
type PieceMsg interface {
	IsPieceMsg()
}

type Square struct {
	piece Piece
}

type Rank struct {
	squares []Square
}

// NewRank creates a Rank from a slice of pieces.
func NewRank(pieces []Piece) Rank {
	squares := make([]Square, len(pieces))
	for i, piece := range pieces {
		squares[i] = Square{
			piece: piece,
		}
	}
	return Rank{squares: squares}
}

// Board represents a 2D grid of squares organized into ranks (rows).
// Board is designed for immutable use in bubbletea/Elm architecture:
// - Navigation methods (MoveUp/Down/Left/Right) return new Board with updated position
// - The underlying ranks slice is shared between Board instances (copy-on-write)
// - This is safe as long as square values are never mutated after board creation
// - If you need to modify square values, clone the ranks slice first
type Board struct {
	ranks    []Rank
	files    []File
	position position
	width    int // Number of files
	height   int // Number of ranks

	// Viewport
	viewportWidth  int
	viewportHeight int
	initialized    bool
	fileOffset     int // Index of leftmost visible file (for horizontal scrolling)

	// Edit mode - when true, keys go to focused piece instead of nav
	editMode bool
}

// Todo: pass style config instead of importing parcours/style
func New(ranks []Rank, files []File, rank, file int) (board Board, err error) {

	board = Board{
		ranks:    ranks,
		files:    files,
		width:    len(files),
		height:   len(ranks),
		position: position{file: file, rank: rank},
	}

	err = board.validate()
	if err != nil {
		return
	}
	board.setPositions()
	return
}

func (brd Board) Replace(ranks []Rank) (board Board, err error) {

	brd.ranks = ranks
	err = brd.validate()
	if err != nil {
		return
	}

	brd.setPositions()
	board = brd
	return
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

func (brd Board) Init() tea.Cmd {
	return nil
}

func (brd Board) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	switch msg := msg.(type) {
	case SizeMsg:
		brd.viewportWidth = msg.Width
		brd.viewportHeight = msg.Height
		brd.initialized = true
		brd.fileOffset = brd.adjustFileOffset()
		return brd, nil
	case ReplaceMsg:
		newBrd, err := brd.Replace(msg.Ranks)
		if err != nil {
			return brd, func() tea.Msg {
				return message.ErrorMsg{Err: err}
			}
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
			return brd.moveUp()
		case "down", "j":
			return brd.moveDown()
		case "left", "h":
			return brd.moveLeft()
		case "right", "l":
			return brd.moveRight()
		case "g":
			return brd.moveTop()
		case "G":
			return brd.moveBottom()
		case "pgup", "ctrl+u":
			return brd.movePageUp()
		case "pgdown", "ctrl+d":
			return brd.movePageDown()
		}
	}

	// Non-key messages still go to focused piece
	return brd.updatePiece(msg)
}

// updatePiece passes a message to the focused piece
func (brd Board) updatePiece(msg tea.Msg) (Board, tea.Cmd) {

	r := brd.position.rank
	f := brd.position.file

	// Note: mutating shared ranks slice!!

	square := brd.ranks[r].squares[f]
	piece, cmd := square.piece.Update(msg)
	brd.ranks[r].squares[f].piece = piece

	return brd, cmd
}

func (brd Board) View() tea.View {
	return brd.viewCanvas()
}

func (brd Board) moveUp() (Board, tea.Cmd) {
	if brd.position.rank > 0 {
		brd.position.rank--
		return brd, brd.positionCmd()
	}
	// Hit top edge
	return brd, func() tea.Msg {
		return NavMsg{Direction: NavUp}
	}
}

func (brd Board) moveDown() (Board, tea.Cmd) {
	if brd.position.rank < brd.height-1 {
		brd.position.rank++
		return brd, brd.positionCmd()
	}
	// Hit bottom edge
	return brd, func() tea.Msg {
		return NavMsg{Direction: NavDown}
	}
}

func (brd Board) moveLeft() (Board, tea.Cmd) {
	if brd.position.file > 0 {
		brd.position.file--
		brd.fileOffset = brd.adjustFileOffset()
		return brd, brd.positionCmd()
	}
	return brd, nil
}

func (brd Board) moveRight() (Board, tea.Cmd) {
	if brd.position.file < brd.width-1 {
		brd.position.file++
		brd.fileOffset = brd.adjustFileOffset()
		return brd, brd.positionCmd()
	}
	return brd, nil
}

func (brd Board) moveTop() (Board, tea.Cmd) {
	// Always move to top of board and signal want absolute top of dataset
	brd.position.rank = 0
	return brd, tea.Batch(
		brd.positionCmd(),
		func() tea.Msg { return NavMsg{Direction: NavTop} },
	)
}

func (brd Board) moveBottom() (Board, tea.Cmd) {
	// Always move to bottom of board and signal want absolute bottom of dataset
	brd.position.rank = brd.height - 1
	return brd, tea.Batch(
		brd.positionCmd(),
		func() tea.Msg { return NavMsg{Direction: NavBottom} },
	)
}

func (brd Board) movePageUp() (Board, tea.Cmd) {
	// Page up: request previous page, preserve cursor position
	return brd, func() tea.Msg {
		return NavMsg{Direction: NavPageUp}
	}
}

func (brd Board) movePageDown() (Board, tea.Cmd) {
	// Page down: request next page, preserve cursor position
	return brd, func() tea.Msg {
		return NavMsg{Direction: NavPageDown}
	}
}

// unexported

type position struct {
	rank int
	file int
}

const (
	headerRow     = 0
	separatorRow  = 1
	dataRowOffset = 2
)

// viewCanvas renders the board using Canvas/Layer composition.
// This is the new rendering approach - call piece.View() and position via Draw().
func (brd Board) viewCanvas() tea.View {
	if !brd.initialized {
		return tea.NewView("loading...")
	}

	canvas := lipgloss.NewCanvas(brd.viewportWidth, brd.viewportHeight)

	// Get visible file range
	visStart, visEnd := brd.visibleFiles()

	// Calculate x positions for each visible file
	xPositions := make([]int, visEnd-visStart)
	x := 0
	for i := visStart; i < visEnd; i++ {
		xPositions[i-visStart] = x
		x += brd.files[i].Width() + gutter
	}

	// Draw headers
	for i := visStart; i < visEnd; i++ {
		file := brd.files[i]
		headerText := fmt.Sprintf("%-*s", file.Width(), file.Name())
		area := image.Rect(xPositions[i-visStart], headerRow, xPositions[i-visStart]+file.Width(), headerRow+1)
		content := uv.NewStyledString(headerText)
		content.Draw(canvas, area)
	}

	// Draw separator line
	separator := strings.Repeat("─", brd.viewportWidth)
	sepContent := uv.NewStyledString(style.TableBorderStyle.Render(separator))
	sepContent.Draw(canvas, image.Rect(0, separatorRow, brd.viewportWidth, separatorRow+1))

	// Pass 1: Draw all piece content
	for r, rank := range brd.ranks {
		y := dataRowOffset + r

		for i := visStart; i < visEnd; i++ {
			f := i - visStart
			square := rank.squares[i]
			fileWidth := brd.files[i].Width()

			pieceView := square.piece.View()
			area := image.Rect(xPositions[f], y, xPositions[f]+fileWidth, y+1)
			pieceView.Content.Draw(canvas, area)
		}
	}

	// Pass 2: Apply highlights by setting cell backgrounds
	// Order matters: col first, then row, then cell (cell wins)
	visualFile := brd.position.file - visStart
	selectedY := dataRowOffset + brd.position.rank

	// Highlight selected column (full height)
	colX := xPositions[visualFile]
	colWidth := brd.files[brd.position.file].Width() + gutter
	for r := range brd.ranks {
		y := dataRowOffset + r
		applyBgToArea(canvas, colX, y, colWidth, style.HlColStyle)
	}

	// Highlight selected row (full viewport width)
	applyBgToArea(canvas, 0, selectedY, brd.viewportWidth, style.HlRowStyle)

	// Highlight selected cell
	applyBgToArea(canvas, colX, selectedY, colWidth, style.HlCellStyle)

	return tea.NewView(canvas)
}

// applyBgToArea sets the background color on all cells in the given area
func applyBgToArea(canvas *lipgloss.Canvas, x, y, width int, sty lipgloss.Style) {
	// Todo: dont style beyond last col
	bg := sty.GetBackground()
	for cx := x; cx < x+width; cx++ {
		cell := canvas.CellAt(cx, y)
		if cell != nil {
			cell.Style.Bg = bg
			canvas.SetCell(cx, y, cell)
		}
	}
}

func truncate(s string, width int) string {
	if width <= 0 {
		return s
	}
	runes := []rune(s)
	if len(runes) <= width {
		return s
	}
	return string(runes[:width-1]) + style.MutedStyle.Render("…")
}

// positionCmd returns a command that sends the current position and cell info
// Todo: consider "delete" etc alternative where board handles certain sensible keys
// and emits a cmd/msg with relevant data, this might be workable??
// the benefit would be not caching this elsewhere
func (brd Board) positionCmd() tea.Cmd {
	pos := SquareMsg{
		Rank:  brd.position.rank,
		File:  brd.position.file,
		Field: brd.files[brd.position.file].Name(),
		Value: brd.ranks[brd.position.rank].squares[brd.position.file].piece.Value(),
	}
	return func() tea.Msg { return pos }
}

// visibleFiles returns the range of files [start, end) that fit in the viewport width
func (brd Board) visibleFiles() (start, end int) {
	return brd.visibleFilesFrom(brd.fileOffset)
}

// visibleFilesFrom returns the range of files [start, end) that fit starting from the given offset
func (brd Board) visibleFilesFrom(fileOffset int) (start, end int) {
	if fileOffset >= len(brd.files) {
		return 0, 0
	}
	if brd.viewportWidth == 0 {
		// No width constraint, show all files from offset
		return fileOffset, len(brd.files)
	}

	start = fileOffset
	usedWidth := 0

	for i := fileOffset; i < len(brd.files); i++ {
		fileWidth := brd.files[i].Width() + gutter
		if usedWidth+fileWidth > brd.viewportWidth {
			break
		}
		usedWidth += fileWidth
		end = i + 1
	}

	return start, end
}

// adjustFileOffset returns fileOffset adjusted to keep position.file visible
func (brd Board) adjustFileOffset() int {
	// If selected file is before visible range, scroll left
	if brd.position.file < brd.fileOffset {
		return brd.position.file
	}

	// If selected file is after visible range, scroll right minimally
	_, visEnd := brd.visibleFiles()
	if brd.position.file >= visEnd {
		// Increment offset until selected file is just visible (at right edge)
		for offset := brd.fileOffset + 1; offset <= brd.position.file; offset++ {
			_, end := brd.visibleFilesFrom(offset)
			if brd.position.file < end {
				return offset
			}
		}
		// Fallback: put selected file at left edge
		return brd.position.file
	}

	return brd.fileOffset
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
