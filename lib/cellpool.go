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
	var continuedCell = Copy(oldCell)
	//TODO: Re-enable when I figure out how to access momentNum
	//Log(LOGTYPE_CELLEFFECT, "cell %d now has a future self established during moment %d\n", oldCell.ID, momentNum)
	oldCell.NextMomentSelf = continuedCell
	return continuedCell
}

func RemoveCellsFromMoment(moment *Moment, cellsToDelete []*Cell) {
	data := make([]*Cell, 0, len(moment.Cells)-len(cellsToDelete))
loop:
	for _, cellThatExists := range moment.Cells {
		for _, cellToDelete := range cellsToDelete {
			if cellThatExists == cellToDelete {
				CellPool.Return(cellToDelete)
				moment.CellsSpatialIndex[cellToDelete.X][cellToDelete.Y] = nil
				continue loop
			}
		}
		data = append(data, cellThatExists)
	}
	moment.Cells = data

}

func Copy(oldCell *Cell) *Cell {
	//TODO: Can this not be done by just a generic struct copy?
	var newCell = CellPool.Borrow()
	newCell.Energy = oldCell.Energy
	newCell.Age = oldCell.Age
	newCell.X = oldCell.X
	newCell.ID = oldCell.ID
	newCell.Y = oldCell.Y
	newCell.SpeciesID = oldCell.SpeciesID
	newCell.PercentChanceWait = oldCell.PercentChanceWait
	newCell.SpeciesColor = oldCell.SpeciesColor
	newCell.EnergySpentOnReproducing = oldCell.EnergySpentOnReproducing
	newCell.Canopy = oldCell.Canopy
	newCell.ClockRate = oldCell.ClockRate
	newCell.GrowCanopyAt = oldCell.GrowCanopyAt
	newCell.EnergyReproduceThreshold = oldCell.EnergyReproduceThreshold
	newCell.Legs = oldCell.Legs
	newCell.MoveChance = oldCell.MoveChance
	newCell.GrowLegsAt = oldCell.GrowLegsAt
	newCell.X_originalMoveChance = oldCell.X_originalMoveChance
	newCell.X_originalGrowLegsAt = oldCell.X_originalGrowLegsAt
	newCell.X_originalEnergyReproduceThreshold = oldCell.X_originalEnergyReproduceThreshold
	newCell.X_originalEnergySpentOnReproducing = oldCell.X_originalEnergySpentOnReproducing
	newCell.X_originalGrowCanopyAt = oldCell.X_originalGrowCanopyAt
	newCell.X_originalPercentChanceWait = oldCell.X_originalPercentChanceWait
	newCell.X_originalClockRate = oldCell.X_originalClockRate

	return newCell
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
