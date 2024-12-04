package in_memory_tile_checker

func New(maxIndex uint32) *Checker {
	return &Checker{
		maxIndex: maxIndex,
	}
}

type Checker struct {
	maxIndex uint32
}

func (c *Checker) CheckTile(tile uint32) bool {
	return tile <= c.maxIndex // uint32 is always >= 0
}

func (c *Checker) MaxIndex() uint32 {
	return c.maxIndex
}