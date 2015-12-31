package lib

var CellPool *Pool

func init() {
	CellPool = NewPool(MAX_CELL_COUNT)
}

var secretIDCounter = 0

func NewPool(max int) *Pool {
	return &Pool{
		pool: make(chan *Cell, max),
	}
}

type Pool struct {
	pool chan *Cell
}

func (p *Pool) Return(c *Cell) {
	select {
	case p.pool <- c:
	default:

	}
}

// Borrow a Cell from the pool.
func (p *Pool) Borrow() *Cell {
	var c *Cell
	select {
	case c = <-p.pool:
	default:
		c = NewCell()
	}
	return c
}

func NewCell() *Cell {
	secretIDCounter++
	return &Cell{_secretID: secretIDCounter}
}
