package lib

//FOUNDATIONAL WORLD VARIABLES

//---RUN_CONDITIONS-----
const RANDOM_SEED = false

const MAX_MOMENTS = 39000

//---WORLD_CONDITIONS---
const SHINE_ENERGY_AMOUNT = 1.0

//const INITIAL_ATMOSPHERIC_MATERIAL = 10.0 * GRID_WIDTH * GRID_HEIGHT

const INITIAL_CELL_COUNT = 2000

const GRID_DEPTH = 4
const GRID_WIDTH = 140
const GRID_HEIGHT = 25

//UI VARIABLES
var LOGTYPES_ENABLED = []int{LOGTYPE_PRINTGRID_GRID, LOGTYPE_PRINTGRID_SUMMARY, LOGTYPE_SPECIESREPORT, LOGTYPE_FINALSTATS, LOGTYPE_ERROR}

var CELLEFFECT_ONLY_IF_TRACED = true

const (
	LOGTYPE_ERROR                  = iota
	LOGTYPE_MAINLOOPSINGLE         = iota
	LOGTYPE_MAINLOOPSINGLE_PRIMARY = iota
	LOGTYPE_FINALSTATS             = iota
	LOGTYPE_HIGHFREQUENCY          = iota
	LOGTYPE_DEBUGCONCURRENCY       = iota
	LOGTYPE_PRINTGRIDCELLS         = iota
	LOGTYPE_PRINTGRID_GRID         = iota
	LOGTYPE_PRINTGRID_BIGGRID      = iota
	LOGTYPE_PRINTGRID_SUMMARY      = iota
	LOGTYPE_OTHER                  = iota
	LOGTYPE_FAILURES               = iota
	LOGTYPE_SPECIALEVENT           = iota
	LOGTYPE_SPECIESREPORT          = iota
	LOGTYPE_CELLEFFECT             = iota
)

const PRINTGRID_EVERY_N_TURNS = 500

const DEFAULT_PRINTGRID_DEPTH = 0

const SPECIES_DIVERGENCE_THRESHOLD = 2.5

const BIGGRID_INCREMENT = 3

const NUM_TOP_SPECIES_TO_PRINT = 6

const ACTUAL_WAIT_MULTIPLIER = 3

const MAX_TRIES_TO_FIND_EMPTY_GRID_COORD = 100

//---PERFORMANCE_VARIABLES---
const CELLACTIONDECIDER_ROUTINECOUNT = 5
const CELLACTIONEXECUTER_ROUTINECOUNT = 1
const NONCELLACTIONDECIDER_ROUTINECOUNT = 1

const MAX_CELL_COUNT = 900000

const CELLS_PER_BUNDLE = 1000

//END CONSTANTS-x-x-x-x-x-x-x-x-x-x-x

//CELL STUFF-----
