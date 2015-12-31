package lib

var WS *WorldState

//---WORLD_CONDITIONS---
const MAX_CELL_COUNT = 90000

const PERCENT_DAYLIGHT = 50 //will be offset by increase shine during day if lowered
const SHINE_ENERGY_AMOUNT = 4.0

const INITIAL_CELL_COUNT = 1000

const GRID_DEPTH = 10
const GRID_WIDTH = 130
const GRID_HEIGHT = 25

const MAX_TRIES_TO_FIND_EMPTY_GRID_COORD = 50

type WorldState struct {
	WSNum                    int
	Cells                    []*Cell
	SpatialIndexSolid        [GRID_DEPTH][GRID_HEIGHT][GRID_WIDTH]*Cell
	SpatialIndexSurfaceCover [GRID_DEPTH][GRID_HEIGHT][GRID_WIDTH]*Cell
}

//TODO: Will need to be revised as move gets more sophisticated
//Not the same as can BE here
func (ws *WorldState) CanMoveHere(cell *Cell, targetX int, targetY int, targetZ int) bool {
	if ws.IsOutOfBounds(targetX, targetY, targetZ) {
		return false
	}
	//var isCovered = ws.IsCovered(targetX, targetY, targetZ)
	//	if isCovered {
	//	return false
	//}

	//Check solid areas
	for i := 0; i < cell.Height+1; i++ {
		var solidCell, solidCellHere = ws.GetSolidCellAt(targetX, targetY, targetZ+i)
		if solidCellHere {
			LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: MOVE LOCATION STATUS %d, %d, %d is occupied by solid cell %d with energy %6.1f\n", cell.ID, targetX, targetY, targetZ+i, solidCell.ID, solidCell.Energy)
			return false
		} else {
			LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: MOVE LOCATION STATUS... Checked %d, %d, %d for solid, but nothing found\n", cell.ID, targetX, targetY, targetZ+i)

		}
	}

	var coveringCell, coveringCellHere = ws.GetCoveringCellAt(targetX, targetY, cell.Z+cell.Height)
	if coveringCellHere {
		LogIfTraced(cell, LOGTYPE_CELLEFFECT, "cell %d: MOVE LOCATION STATUS %d, %d, %d is occupied by covering cell %d with energy %6.1f\n", cell.ID, targetX, targetY, cell.Z+cell.Height, coveringCell.ID, coveringCell.Energy)
		return false
	}

	return true
}

//TODO: Should call both covered and solid instead of doing it itself
func (worldState *WorldState) GetAnyCellAt(x, y, z int) (*Cell, bool) {
	if worldState.IsOutOfBounds(x, y, z) {
		return nil, false
	}

	var solidAt = worldState.SpatialIndexSolid[z][y][x]
	var coveredAt = worldState.SpatialIndexSurfaceCover[z][y][x]

	//TODO: Potential for weird bug with conflict on surfaceCover and solid here
	if solidAt != nil {
		return solidAt, true
	} else if coveredAt != nil {
		return coveredAt, true
	} else {
		return nil, false
	}
}

func (worldState *WorldState) GetCoveringCellAt(x, y, z int) (*Cell, bool) {
	if worldState.IsOutOfBounds(x, y, z) {
		return nil, false
	}

	var coveredAt = worldState.SpatialIndexSurfaceCover[z][y][x]

	if coveredAt != nil {
		return coveredAt, true
	} else {
		return nil, false
	}
}

func (worldState *WorldState) GetSolidCellAt(x, y, z int) (*Cell, bool) {
	if worldState.IsOutOfBounds(x, y, z) {
		return nil, false
	}

	var solidAt = worldState.SpatialIndexSolid[z][y][x]

	if solidAt != nil {
		return solidAt, true
	} else {
		return nil, false
	}
}

func (worldState *WorldState) AddCellToSpatialIndex(cell *Cell) {
	//Cover-Surface == if not legs, Cover-Ground at z + height
	//Non-Solid == 1 around for canopy on same z level, and 1 in current x,y,z
	//Solid == 1 for each height, starting with Z level.
	//fmt.Printf("%d %d %d\n", cell.Z+i, cell.Y, cell.X)
	if cell.Height >= 1 {
		for i := 0; i < cell.Height; i++ {
			WS.SpatialIndexSolid[cell.Z+i][cell.Y][cell.X] = cell
			Log(LOGTYPE_HIGHFREQUENCY, "Adding cell %d to %d,%d,%d in solid index\n", cell.ID, cell.X, cell.Y, cell.Z+i)
		}
	}
	Log(LOGTYPE_HIGHFREQUENCY, "Adding cell %d to %d,%d,%d in surface cover\n", cell.ID, cell.X, cell.Y, cell.Z+cell.Height)
	WS.SpatialIndexSurfaceCover[cell.Z+cell.Height][cell.Y][cell.X] = cell
}

func (worldState *WorldState) RemoveCellFromSpatialIndex(cell *Cell) {
	if cell.Height >= 1 {
		for i := 0; i < cell.Height; i++ {
			WS.SpatialIndexSolid[cell.Z+i][cell.Y][cell.X] = nil
		}
	}
	WS.SpatialIndexSurfaceCover[cell.Z+cell.Height][cell.Y][cell.X] = nil
}

func (worldState *WorldState) IsSolidOrCovered(x int, y int, z int) bool {
	if WS.IsOutOfBounds(x, y, z) {
		return true
	}
	//TODO: Might want to just out and out make it 3D at some point
	return WS.IsSolid(x, y, z) || WS.IsCovered(x, y, z)
}

func (worldState *WorldState) IsCovered(x, y, z int) bool {
	return WS.IsOutOfBounds(x, y, z) || WS.SpatialIndexSurfaceCover[z][y][x] != nil
}

func (worldState *WorldState) IsSolid(x, y, z int) bool {
	return WS.IsOutOfBounds(x, y, z) || WS.SpatialIndexSolid[z][y][x] != nil
}

func (worldState *WorldState) ReturnCellsToPool() {
	for ci := range WS.Cells {
		var cellToReturn = WS.Cells[ci]
		CellPool.Return(cellToReturn)
	}
}

func (worldState *WorldState) IsOutOfBounds(x int, y int, z int) bool {
	return x < 0 || x > GRID_WIDTH-1 || y < 0 || y > GRID_HEIGHT-1 || z < 0 || z > GRID_DEPTH-1
}

//In the plane of z
func (worldState *WorldState) getSurroundingCells(x int, y int, z int) []*Cell {
	var surroundingCells []*Cell
	surroundingCells = make([]*Cell, 0, 9)
	for relativeX := -1; relativeX < 2; relativeX++ {
		for relativeY := -1; relativeY < 2; relativeY++ {
			var xTry = x + relativeX
			var yTry = y + relativeY
			//Inlined from !outofBounds and IsSolidOrCovered
			if !(xTry < 0 || xTry > GRID_WIDTH-1 || yTry < 0 || yTry > GRID_HEIGHT-1 || z < 0 || z > GRID_DEPTH) && WS.SpatialIndexSurfaceCover[z][y][x] != nil {
				var cell = WS.SpatialIndexSurfaceCover[z][yTry][xTry]
				surroundingCells = append(surroundingCells, cell)
			}
		}
	}
	return surroundingCells
}
