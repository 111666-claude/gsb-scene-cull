// Package scenecull 是场景剔除：按空间网格与可视半径剔除，结果不漏且有序。
// 缺陷：查询全量扫描所有对象、范围用方形而不是圆形、结果不按距离排序、同一 ID 会重复返回。
package scenecull

// CellSize 是空间网格的格子边长。
const CellSize = 10.0

// Object 是场景里的一个对象。
type Object struct {
	ID string
	X  float64
	Y  float64
}

// Culler 是剔除器。
type Culler struct {
	objects []Object
	scanned int
}

// NewCuller 建剔除器。
func NewCuller(objects []Object) *Culler {
	return &Culler{objects: append([]Object(nil), objects...)}
}

// Visible 返回可视范围内的对象 ID。
// 缺陷：全量扫描所有对象（不用网格）、用方形范围代替圆形（角落误判）、
// 按录入顺序返回（不排序）、同一 ID 会重复出现。
func (c *Culler) Visible(viewX, viewY, radius float64) []string {
	out := make([]string, 0, 8)
	for _, object := range c.objects {
		c.scanned++
		dx := object.X - viewX
		dy := object.Y - viewY
		if dx <= radius && dx >= -radius && dy <= radius && dy >= -radius {
			out = append(out, object.ID)
		}
	}
	return out
}

// Scanned 是累计扫描的对象数（规模观测）。
func (c *Culler) Scanned() int { return c.scanned }
