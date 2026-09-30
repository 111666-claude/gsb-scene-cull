// Package scenecull 是场景剔除：按空间网格与可视半径剔除，结果不漏且有序。
package scenecull

import (
	"math"
	"sort"
)

// CellSize 是空间网格的格子边长。
const CellSize = 10.0

// Object 是场景里的一个对象。
type Object struct {
	ID string
	X  float64
	Y  float64
}

// cellKey 是网格格子的坐标。
type cellKey struct {
	x int
	y int
}

// Culler 是剔除器，按 CellSize 网格索引对象。
type Culler struct {
	cells   map[cellKey][]Object
	scanned int
}

// NewCuller 建剔除器，把对象放进所在格子。
func NewCuller(objects []Object) *Culler {
	culler := &Culler{cells: make(map[cellKey][]Object)}
	for _, object := range objects {
		key := cellKey{cellIndex(object.X), cellIndex(object.Y)}
		culler.cells[key] = append(culler.cells[key], object)
	}
	return culler
}

// cellIndex 返回坐标所在的格子序号（负坐标也正确向下取整）。
func cellIndex(coord float64) int {
	return int(math.Floor(coord / CellSize))
}

// hit 是一次命中：对象 ID 与到视点的距离平方。
type hit struct {
	id   string
	dist float64
}

// Visible 返回可视范围内的对象 ID：只扫可视范围覆盖的格子，
// 按圆形判定（dx*dx+dy*dy <= radius*radius），按距离升序、同距按 ID 升序，
// 同一 ID 只出现一次（保留最近的距离）。
func (c *Culler) Visible(viewX, viewY, radius float64) []string {
	if radius < 0 {
		return nil
	}
	radiusSq := radius * radius
	minX, maxX := cellIndex(viewX-radius), cellIndex(viewX+radius)
	minY, maxY := cellIndex(viewY-radius), cellIndex(viewY+radius)
	hits := make([]hit, 0, 8)
	slots := make(map[string]int)
	for cx := minX; cx <= maxX; cx++ {
		for cy := minY; cy <= maxY; cy++ {
			for _, object := range c.cells[cellKey{cx, cy}] {
				c.scanned++
				dx := object.X - viewX
				dy := object.Y - viewY
				dist := dx*dx + dy*dy
				if dist > radiusSq {
					continue
				}
				if slot, ok := slots[object.ID]; ok {
					if dist < hits[slot].dist {
						hits[slot].dist = dist
					}
					continue
				}
				slots[object.ID] = len(hits)
				hits = append(hits, hit{id: object.ID, dist: dist})
			}
		}
	}
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].dist != hits[j].dist {
			return hits[i].dist < hits[j].dist
		}
		return hits[i].id < hits[j].id
	})
	out := make([]string, len(hits))
	for index, h := range hits {
		out[index] = h.id
	}
	return out
}

// Scanned 是累计扫描的对象数（规模观测）。
func (c *Culler) Scanned() int { return c.scanned }
