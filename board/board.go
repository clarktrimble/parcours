package board

import (
	"image"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/pkg/errors"

	"parcours/message"
	"parcours/style"
)

const (
	gutter    = 1 // space between columns
	separator = "-"
)

type File interface {
	Name() string
	Width() int
}

// Piece represents a board piece that can update and render itself.
type Piece interface {
	Update(tea.Msg) (Piece, tea.Cmd)
	View() tea.View
	Value() string // Returns the raw value (for filtering, etc.) Todo: nt.Value ??
	// Todo: want pieces to be tea.Model, but what about Value()??
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
	// Todo: Rank _is_ []Square ??
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
	vpw         int
	vph         int
	initialized bool
	fileOffset  int // Index of leftmost visible file (for horizontal scrolling)

	// Layout cache - updated when fileOffset or viewport changes
	layoutAreas []image.Rectangle
	layoutEnd   int // Index (exclusive) of last file to draw

	// Edit mode - when true, keys go to focused piece instead of nav
	editMode bool
}

// New creates a Board from rank and file.
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

// replace replaces a board's ranks.
func (brd Board) replace(ranks []Rank) (board Board, err error) {

	brd.ranks = ranks
	err = brd.validate()
	if err != nil {
		return
	}

	brd.setPositions()
	board = brd
	return
}

func (brd Board) Init() tea.Cmd {
	return nil
}

func (brd Board) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	switch msg := msg.(type) {
	case SizeMsg:
		brd.vpw = msg.Width
		brd.vph = msg.Height
		brd.initialized = true
		return brd.checkFileOffset(), nil
	case ReplaceMsg:
		newBrd, err := brd.replace(msg.Ranks)
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
	if !brd.initialized {
		return tea.NewView("loading...")
	}

	canvas := lipgloss.NewCanvas(brd.vpw, 2+len(brd.ranks)) // Todo: demagic

	files := brd.files[brd.fileOffset:brd.layoutEnd]
	drawHeaders(canvas, files, brd.layoutAreas)

	// Shift areas to data row (after header + separator)
	areas := slices.Clone(brd.layoutAreas)
	nextY := brd.layoutAreas[0].Max.Y + 1 // Todo: dont crash
	for i := range areas {
		areas[i].Min.Y = nextY
		areas[i].Max.Y = nextY + 1
	}
	drawRanks(canvas, brd.ranks, brd.fileOffset, brd.layoutEnd, areas)
	brd.drawHighlight(canvas, areas)

	return tea.NewView(canvas)
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
		brd = brd.checkFileOffset()
		return brd, brd.positionCmd()
	}
	return brd, nil
}

func (brd Board) moveRight() (Board, tea.Cmd) {
	if brd.position.file < brd.width-1 {
		brd.position.file++
		brd = brd.checkFileOffset()
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

func (brd *Board) setPositions() {
	for r, rank := range brd.ranks {
		for f := range rank.squares {
			piece := rank.squares[f].piece
			updated, _ := piece.Update(PositionMsg{Rank: r, File: f})
			brd.ranks[r].squares[f].piece = updated
		}
	}
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

// Note: mutates shared slice, safe with Elm pattern (caller replaces copy)
func (brd Board) updateLayout() Board {
	brd.layoutAreas = brd.layoutAreas[:0]
	x := 0
	for i := brd.fileOffset; i < len(brd.files); i++ {
		if x >= brd.vpw {
			break
		}
		w := brd.files[i].Width()
		brd.layoutAreas = append(brd.layoutAreas, image.Rect(x, 0, x+w, 1))
		x += w + gutter
		brd.layoutEnd = i + 1
	}
	return brd
}

func (brd Board) drawHighlight(canvas *lipgloss.Canvas, areas []image.Rectangle) {
	selectedY := areas[0].Min.Y + brd.position.rank // Todo: dont crash
	visualFile := brd.position.file - brd.fileOffset
	cellArea := areas[visualFile]

	rankArea := fullSpan(areas, selectedY)
	applyBgToArea(canvas, rankArea.Min.X, rankArea.Min.Y, rankArea.Dx(), style.HlRowStyle)
	applyBgToArea(canvas, cellArea.Min.X, selectedY, cellArea.Dx(), style.HlCellStyle)
}

func fullSpan(areas []image.Rectangle, y int) image.Rectangle {
	if len(areas) == 0 {
		return image.Rectangle{}
	}
	return image.Rect(areas[0].Min.X, y, areas[len(areas)-1].Max.X, y+1)
}

func drawHeaders(canvas *lipgloss.Canvas, files []File, areas []image.Rectangle) {
	for i, file := range files {
		// Todo: consider adding View() to File
		uv.NewStyledString(file.Name()).Draw(canvas, areas[i])
	}

	area := fullSpan(areas, areas[0].Max.Y)
	str := strings.Repeat(separator, area.Dx())
	uv.NewStyledString(style.TableBorderStyle.Render(str)).Draw(canvas, area)
}

func drawRank(canvas *lipgloss.Canvas, squares []Square, areas []image.Rectangle) {
	for i, square := range squares {
		square.piece.View().Content.Draw(canvas, areas[i])
	}
}

func drawRanks(canvas *lipgloss.Canvas, ranks []Rank, start, end int, areas []image.Rectangle) {
	rectangles := slices.Clone(areas)
	for _, rank := range ranks {
		drawRank(canvas, rank.squares[start:end], rectangles)
		for i := range rectangles {
			rectangles[i].Min.Y++
			rectangles[i].Max.Y++
		}
	}
}

// Todo: save this func til we find a home for it in linepanel or value piece
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

func (brd Board) checkFileOffset() Board {
	if brd.position.file < brd.fileOffset {
		brd.fileOffset = brd.position.file
		return brd.updateLayout()
	}
	if brd.position.file >= brd.layoutEnd {
		for offset := brd.fileOffset + 1; offset <= brd.position.file; offset++ {
			if brd.fileVisibleFrom(offset, brd.position.file) {
				brd.fileOffset = offset
				return brd.updateLayout()
			}
		}
		brd.fileOffset = brd.position.file
		return brd.updateLayout()
	}
	return brd.updateLayout()
}

func (brd Board) fileVisibleFrom(offset, file int) bool {
	// Todo: can be more efficient? (find last visible?)
	x := 0
	for i := offset; i <= file; i++ {
		if x+brd.files[i].Width() > brd.vpw {
			return false
		}
		x += brd.files[i].Width() + gutter
	}
	return true
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
