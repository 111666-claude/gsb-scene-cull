// Package scenecull 是场景剔除：按空间网格与可视半径剔除，结果不漏且有序。
//
// 对象按 CellSize 划分到均匀网格；一次查询只枚举与可视圆相交的格子，
// 对其中的对象做圆形判定（dx*dx + dy*dy <= radius*radius），
// 结果按距离升序、同距按 ID 升序返回，同一 ID 只出现一次（取最近位置）。
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

type cellKey struct {
	cx int
	cy int
}

type hit struct {
	id string
	d2 float64
}

// Culler 是剔除器。
type Culler struct {
	objects []Object
	cells   map[cellKey][]int
	scanned int
	seen    map[string]int
	hits    []hit
}

// NewCuller 建剔除器，并把对象登记到 CellSize 网格里。
func NewCuller(objects []Object) *Culler {
	copied := append([]Object(nil), objects...)
	c := &Culler{
		objects: copied,
		cells:   make(map[cellKey][]int, len(copied)/4+1),
		seen:    make(map[string]int, 16),
	}
	for index := range copied {
		key := c.keyAt(copied[index].X, copied[index].Y)
		c.cells[key] = append(c.cells[key], index)
	}
	return c
}

func (c *Culler) keyAt(x, y float64) cellKey {
	return cellKey{
		cx: int(math.Floor(x / CellSize)),
		cy: int(math.Floor(y / CellSize)),
	}
}

// Visible 返回可视范围内的对象 ID：按距离升序、同距按 ID 升序，同一 ID 只出现一次。
// 只扫描与可视圆相交的格子；半径为负时视为没有可视对象。
func (c *Culler) Visible(viewX, viewY, radius float64) []string {
	c.hits = c.hits[:0]
	if radius < 0 {
		return nil
	}
	r2 := radius * radius

	// 圆外接包围盒给出候选格子范围；逐个格子再做圆-格子相交判定，
	// 保证只扫真正被圆覆盖到的格子（角落格子不含在内）。
	minX, maxX := viewX-radius, viewX+radius
	minY, maxY := viewY-radius, viewY+radius
	epsilon := 1e-9 * math.Max(1.0, math.Abs(radius))
	x0 := int(math.Floor(minX / CellSize))
	x1 := int(math.Floor(maxX / CellSize))
	y0 := int(math.Floor(minY / CellSize))
	y1 := int(math.Floor((maxY + epsilon) / CellSize))

	for cx := x0; cx <= x1; cx++ {
		cellMinX := float64(cx) * CellSize
		cellMaxX := cellMinX + CellSize
		for cy := y0; cy <= y1; cy++ {
			cellMinY := float64(cy) * CellSize
			cellMaxY := cellMinY + CellSize
			if !circleIntersectsRect(viewX, viewY, r2, cellMinX, cellMinY, cellMaxX, cellMaxY, epsilon) {
				continue
			}
			for _, index := range c.cells[cellKey{cx: cx, cy: cy}] {
				c.scanned++
				object := c.objects[index]
				dx := object.X - viewX
				dy := object.Y - viewY
				d2 := dx*dx + dy*dy
				if d2 > r2 {
					continue
				}
				if pos, ok := c.seen[object.ID]; ok {
					if d2 < c.hits[pos].d2 {
						c.hits[pos].d2 = d2
					}
					continue
				}
				c.seen[object.ID] = len(c.hits)
				c.hits = append(c.hits, hit{id: object.ID, d2: d2})
			}
		}
	}
	for index := range c.hits {
		delete(c.seen, c.hits[index].id)
	}

	sort.SliceStable(c.hits, func(i, j int) bool {
		if c.hits[i].d2 != c.hits[j].d2 {
			return c.hits[i].d2 < c.hits[j].d2
		}
		return c.hits[i].id < c.hits[j].id
	})

	out := make([]string, len(c.hits))
	for index := range c.hits {
		out[index] = c.hits[index].id
	}
	return out
}

// circleIntersectsRect 判断圆是否与轴对齐矩形相交（含边界），矩形外的圆用最近点距离判定。
func circleIntersectsRect(cx, cy, r2, minX, minY, maxX, maxY, epsilon float64) bool {
	nearestX := 0.0
	if cx < minX {
		nearestX = minX - cx
	} else if cx > maxX {
		nearestX = cx - maxX
	}
	nearestY := 0.0
	if cy < minY {
		nearestY = minY - cy
	} else if cy > maxY {
		nearestY = cy - maxY
	}
	return nearestX*nearestX+nearestY*nearestY <= r2+epsilon
}

// Scanned 是累计扫描的对象数（规模观测）。
func (c *Culler) Scanned() int { return c.scanned }
