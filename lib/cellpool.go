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

//TODO: ContinueOn and Copy would preferably be in cell, but can't be for reasons of depending on co
func (oldCell *Cell) ContinueOn() *Cell {
	//	var continuedCell = //Copy(oldCell)
	//TODO: Re-enable when I figure out how to access WSNum
	//Log(LOGTYPE_CELLEFFECT, "cell %d now has a future self established during WS %d\n", oldCell.ID, WSNum)
	//if TracedCell != nil && oldCell.ID == TracedCell.ID {
	//	TracedCell = continuedCell
	//}
	//TODO: This whole thing should be removed eventually, but it's not yet
	return oldCell
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
