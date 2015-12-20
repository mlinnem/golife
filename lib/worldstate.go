package lib

var WS *WorldState

//---WORLD_CONDITIONS---
const MAX_CELL_COUNT = 90000

const SHINE_ENERGY_AMOUNT = 1.0

const INITIAL_CELL_COUNT = 2000

const GRID_DEPTH = 2
const GRID_WIDTH = 140
const GRID_HEIGHT = 25

const MAX_TRIES_TO_FIND_EMPTY_GRID_COORD = 100

type WorldState struct {
	WSNum                    int
	Cells                    []*Cell
	SpatialIndexSolid        [GRID_DEPTH][GRID_HEIGHT][GRID_WIDTH]*Cell
	SpatialIndexSurfaceCover [GRID_DEPTH][GRID_HEIGHT][GRID_WIDTH]*Cell
}

//TODO: Will need to be revised as move gets more sophisticated
func (ws *WorldState) CanMoveHere(cell *Cell, targetX int, targetY int, targetZ int) bool {
	var coverAreaIsOpen = ws.IsCovered(targetX, targetY, targetZ)
	if !coverAreaIsOpen {
		return false
	}

	//Check solid areas
	//Note: this wont run at all if height is 0
	for i := 0; i < cell.Height; i++ {
		if ws.IsSolid(targetX, targetY, targetZ) {
			return false
		}
	}

	return true
}

func (worldState *WorldState) AddCellToSpatialIndex(cell *Cell) {
	//Cover-Surface == if not legs, Cover-Ground at z + height
	//Non-Solid == 1 around for canopy on same z level, and 1 in current x,y,z
	//Solid == 1 for each height, starting with Z level.
	//fmt.Printf("%d %d %d\n", cell.Z+i, cell.Y, cell.X)
	if cell.Height >= 1 {
		for i := 0; i < cell.Height; i++ {
			WS.SpatialIndexSolid[cell.Z+i][cell.Y][cell.X] = cell
			//	Log(LOGTYPE_HIGHFREQUENCY, "Adding cell %d to %d,%d,%d in solid index\n", cell.ID, cell.X, cell.Y, cell.Z)
		}
	}
	//	Log(LOGTYPE_HIGHFREQUENCY, "Adding cell %d to %d,%d,%d in surface cover\n", cell.ID, cell.X, cell.Y, cell.Z+cell.Height)
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
