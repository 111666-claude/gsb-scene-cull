package scenecull

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"testing"
	"time"
)

// 圆形边界：角落实测在方形内、圆形外，必须剔除；圆周上的点算可见。
func TestCircleBoundary(t *testing.T) {
	objects := []Object{
		{ID: "corner-pp", X: 8, Y: 8},
		{ID: "corner-np", X: -8, Y: 8},
		{ID: "corner-pn", X: 8, Y: -8},
		{ID: "corner-nn", X: -8, Y: -8},
		{ID: "axis-x", X: 10, Y: 0},
		{ID: "axis-y", X: 0, Y: -10},
		{ID: "on-circle", X: 6, Y: 8},
		{ID: "inside", X: 7, Y: 7},
	}
	got := NewCuller(objects).Visible(0, 0, 10)
	want := []string{"inside", "axis-x", "axis-y", "on-circle"}
	assertIDs(t, got, want)
}

// 结果有序：距离升序，同距按 ID 升序，与录入顺序无关。
func TestResultOrder(t *testing.T) {
	objects := []Object{
		{ID: "far", X: 3, Y: 4},   // 距离 5
		{ID: "near", X: 1, Y: 0},  // 距离 1
		{ID: "tie-b", X: 3, Y: 4}, // 距离 5
		{ID: "mid", X: 0, Y: 3},   // 距离 3
		{ID: "tie-a", X: 4, Y: 3}, // 距离 5
	}
	got := NewCuller(objects).Visible(0, 0, 5)
	want := []string{"near", "mid", "far", "tie-a", "tie-b"}
	assertIDs(t, got, want)
}

// 同一 ID（位置不同或完全重复）只出现一次，并按最近距离排位。
func TestDuplicateID(t *testing.T) {
	objects := []Object{
		{ID: "dup", X: 4, Y: 0}, // 距离 4
		{ID: "dup", X: 1, Y: 0}, // 距离 1，更近
		{ID: "dup", X: 1, Y: 0}, // 完全重复
		{ID: "other", X: 2, Y: 0},
	}
	got := NewCuller(objects).Visible(0, 0, 5)
	want := []string{"dup", "other"}
	assertIDs(t, got, want)
}

// 查询代价：只扫覆盖格子里的对象，远处的对象不计入 Scanned。
func TestScannedOnlyCoveredCells(t *testing.T) {
	objects := []Object{
		{ID: "same-cell-outside-circle", X: 9, Y: 9},
		{ID: "hit", X: 1, Y: 1},
	}
	for index := 0; index < 10000; index++ {
		objects = append(objects, Object{
			ID: fmt.Sprintf("far-%d", index),
			X:  10000 + float64(index),
			Y:  10000,
		})
	}
	culler := NewCuller(objects)
	before := culler.Scanned()
	got := culler.Visible(0, 0, 10)
	delta := culler.Scanned() - before
	covered := objectsInCoveredCells(objects, 0, 0, 10)
	if delta != covered {
		t.Fatalf("扫描数应等于覆盖格内对象数：scanned=%d covered=%d", delta, covered)
	}
	if delta > covered+len(got) {
		t.Fatalf("scanned 超过覆盖格对象数加命中数：%d > %d+%d", delta, covered, len(got))
	}
	if len(got) != 1 || got[0] != "hit" {
		t.Fatalf("只有命中的对象可见，得到 %v", got)
	}
}

// 半径不覆盖远处对象时，大量远处对象不应产生任何扫描。
func TestScannedZeroWhenCellsEmpty(t *testing.T) {
	objects := make([]Object, 0, 50000)
	for index := 0; index < 50000; index++ {
		objects = append(objects, Object{ID: fmt.Sprintf("o-%d", index), X: 10000, Y: 10000})
	}
	culler := NewCuller(objects)
	for query := 0; query < 1000; query++ {
		culler.Visible(0, 0, 10)
	}
	if scanned := culler.Scanned(); scanned > 6000 {
		t.Fatalf("扫了 %d 个远处对象，预期 0", scanned)
	}
}

// 重复查询结果相同。
func TestRepeatedQueryDeterministic(t *testing.T) {
	objects := []Object{
		{ID: "a", X: 3, Y: 4},
		{ID: "a", X: 1, Y: 1},
		{ID: "b", X: 2, Y: 2},
		{ID: "c", X: 5, Y: 0},
	}
	culler := NewCuller(objects)
	first := culler.Visible(0, 0, 5)
	for query := 0; query < 10; query++ {
		assertIDs(t, culler.Visible(0, 0, 5), first)
	}
}

// 规模：20 万对象、10 万次查询，单次只付覆盖格代价，整体秒级。
func TestScale200kObjects100kQueries(t *testing.T) {
	const count = 200000
	objects := make([]Object, 0, count)
	for index := 0; index < count; index++ {
		objects = append(objects, Object{
			ID: fmt.Sprintf("o-%d", index),
			X:  float64(index%450)*CellSize + CellSize/2,
			Y:  float64(index/450)*CellSize + CellSize/2,
		})
	}
	culler := NewCuller(objects)
	viewX, viewY, radius := 1005.0, 25.0, 5.0
	const queries = 100000
	start := time.Now()
	var hits []string
	for query := 0; query < queries; query++ {
		hits = culler.Visible(viewX, viewY, radius)
	}
	elapsed := time.Since(start)
	if elapsed > 2*time.Second {
		t.Fatalf("10 万次查询耗时 %v，超过 2s", elapsed)
	}
	covered := objectsInCoveredCells(objects, viewX, viewY, radius)
	if scanned := culler.Scanned(); scanned > queries*(covered+len(hits)) {
		t.Fatalf("扫描数 %d 超过 %d 次查询 × (覆盖格对象 %d + 命中 %d)",
			scanned, queries, covered, len(hits))
	}
}

// 对照自证：200 组随机场景与参考实现（全量判定 + 圆形 + 排序去重）逐项一致。
func TestMatchesReference200RandomCases(t *testing.T) {
	rng := rand.New(rand.NewSource(20260930))
	for caseIndex := 0; caseIndex < 200; caseIndex++ {
		n := rng.Intn(300)
		objects := make([]Object, n)
		idPool := 1 + rng.Intn(n+2)
		for index := range objects {
			x := rng.Float64()*120 - 60
			y := rng.Float64()*120 - 60
			if rng.Intn(5) == 0 {
				x = math.Round(x)
				y = math.Round(y)
			}
			objects[index] = Object{
				ID: fmt.Sprintf("id-%d", rng.Intn(idPool)),
				X:  x,
				Y:  y,
			}
		}
		viewX := rng.Float64()*80 - 40
		viewY := rng.Float64()*80 - 40
		radius := rng.Float64() * 30
		if rng.Intn(8) == 0 {
			radius = 0
		}
		got := NewCuller(append([]Object(nil), objects...)).Visible(viewX, viewY, radius)
		want := referenceVisible(objects, viewX, viewY, radius)
		if len(got) != len(want) {
			t.Fatalf("用例 %d：数量不一致 got=%v want=%v", caseIndex, got, want)
		}
		for index := range got {
			if got[index] != want[index] {
				t.Fatalf("用例 %d：顺序/集合不一致\n got=%v\nwant=%v", caseIndex, got, want)
			}
		}
	}
}

// referenceVisible 是参考实现：全量两两判定 + 圆形范围 + 距离/ID 排序 + 去重。
func referenceVisible(objects []Object, viewX, viewY, radius float64) []string {
	if radius < 0 {
		return nil
	}
	radiusSq := radius * radius
	type refHit struct {
		id   string
		dist float64
	}
	hits := make([]refHit, 0)
	slots := make(map[string]int)
	for _, object := range objects {
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
		hits = append(hits, refHit{id: object.ID, dist: dist})
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

// objectsInCoveredCells 独立统计可视范围覆盖格子里的对象数。
func objectsInCoveredCells(objects []Object, viewX, viewY, radius float64) int {
	minX := int(math.Floor((viewX - radius) / CellSize))
	maxX := int(math.Floor((viewX + radius) / CellSize))
	minY := int(math.Floor((viewY - radius) / CellSize))
	maxY := int(math.Floor((viewY + radius) / CellSize))
	count := 0
	for _, object := range objects {
		cx := int(math.Floor(object.X / CellSize))
		cy := int(math.Floor(object.Y / CellSize))
		if cx >= minX && cx <= maxX && cy >= minY && cy <= maxY {
			count++
		}
	}
	return count
}

func assertIDs(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("结果数量不一致：got=%v want=%v", got, want)
	}
	for index := range got {
		if got[index] != want[index] {
			t.Fatalf("结果不一致：got=%v want=%v", got, want)
		}
	}
}
