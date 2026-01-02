package board

import (
	"image"
	"slices"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/pkg/errors"

	"parcours/style"
)

const (
	gutter       = 1  // space between columns
	separator    = "-"
	headerHeight = 2  // header row + separator line
	jumpBy       = 10 // ranks to scroll with alt+up/down

	// Scroll acceleration
	accelWindow = 200 * time.Millisecond // reset if no repeat within this window
	accelMax    = 10                     // max step size
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

// Board represents a grid of squares with ranks and files.
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

	// Scroll acceleration
	repeatDir   string
	repeatTime  time.Time
	repeatCount int
}

// New creates a Board with viewport size.
// Todo: pass style config instead of importing parcours/style
func New(ranks []Rank, files []File, rank, file, vpw, vph int) (Board, error) {
	brd := Board{
		ranks:       ranks,
		files:       files,
		width:       len(files),
		height:      len(ranks),
		position:    position{file: file, rank: rank},
		vpw:         vpw,
		vph:         vph,
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
		brd.vph = msg.Height
		brd.initialized = true
		return brd.checkFileOffset(), nil

	case ReplaceMsg:
		newBrd, err := brd.replace(msg.Ranks)
		if err != nil {
			return brd, func() tea.Msg { return err }
		}
		return newBrd, newBrd.positionCmd()

		/*
			case AppendMsg:
				return brd.appendRank(msg.Rank)

			case RemoveMsg:
				return brd.removeRank(msg.Index)
		*/

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

/*
func (brd Board) appendRank(rank Rank) (Board, tea.Cmd) {
	brd.ranks = append(brd.ranks, rank)
	brd.height++
	brd.setPositionForRank(brd.height - 1)
	return brd, nil
}

func (brd Board) removeRank(index int) (Board, tea.Cmd) {
	if index < 0 || index >= brd.height {
		return brd, nil
	}
	brd.ranks = append(brd.ranks[:index], brd.ranks[index+1:]...)
	brd.height--
	// Adjust cursor if needed
	if brd.position.rank >= brd.height && brd.height > 0 {
		brd.position.rank = brd.height - 1
	}
	if brd.height == 0 {
		brd.position.rank = 0
	}
	// Reindex positions for ranks after deleted one
	for r := index; r < brd.height; r++ {
		brd.setPositionForRank(r)
	}
	return brd, brd.positionCmd()
}

func (brd *Board) setPositionForRank(r int) {
	for f := range brd.ranks[r].squares {
		piece := brd.ranks[r].squares[f].piece
		updated, _ := piece.Update(PositionMsg{Rank: r, File: f})
		brd.ranks[r].squares[f].piece = updated
	}
}
*/

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

func (brd Board) updateLayout() Board {
	brd.layoutAreas = brd.layoutAreas[:0]
	x := 0
	for i := brd.fileOffset; i < len(brd.files); i++ {
		if x >= brd.vpw {
			break
		}
		w := brd.files[i].Width
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
		uv.NewStyledString(file.Name).Draw(canvas, areas[i])
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
		if x+brd.files[i].Width > brd.vpw {
			return false
		}
		x += brd.files[i].Width + gutter
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
