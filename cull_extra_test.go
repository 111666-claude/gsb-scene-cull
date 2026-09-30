package scenecull

import (
	"math"
	"math/rand"
	"reflect"
	"sort"
	"testing"
)

// bruteVisible 是「全量两两判定」参考实现：圆形判定 + 同 ID 取最近 + 距离/ID 排序。
func bruteVisible(objects []Object, viewX, viewY, radius float64) []string {
	if radius < 0 {
		return []string{}
	}
	r2 := radius * radius
	best := make(map[string]float64)
	for _, object := range objects {
		dx := object.X - viewX
		dy := object.Y - viewY
		d2 := dx*dx + dy*dy
		if d2 > r2 {
			continue
		}
		if previous, ok := best[object.ID]; !ok || d2 < previous {
			best[object.ID] = d2
		}
	}
	type entry struct {
		id string
		d2 float64
	}
	entries := make([]entry, 0, len(best))
	for id, d2 := range best {
		entries = append(entries, entry{id: id, d2: d2})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].d2 != entries[j].d2 {
			return entries[i].d2 < entries[j].d2
		}
		return entries[i].id < entries[j].id
	})
	out := make([]string, len(entries))
	for index := range entries {
		out[index] = entries[index].id
	}
	return out
}

// coveredObjectCount 统计与可视圆相交的格子里一共有多少对象（参考口径）。
func coveredObjectCount(objects []Object, viewX, viewY, radius float64) int {
	epsilon := 1e-9 * math.Max(1.0, math.Abs(radius))
	r2 := radius * radius
	x0 := int(math.Floor((viewX - radius) / CellSize))
	x1 := int(math.Floor((viewX + radius) / CellSize))
	y0 := int(math.Floor((viewY - radius) / CellSize))
	y1 := int(math.Floor((viewY + radius + epsilon) / CellSize))
	covered := make(map[cellKey]bool)
	for cx := x0; cx <= x1; cx++ {
		cellMinX := float64(cx) * CellSize
		cellMaxX := cellMinX + CellSize
		for cy := y0; cy <= y1; cy++ {
			cellMinY := float64(cy) * CellSize
			cellMaxY := cellMinY + CellSize
			if circleIntersectsRect(viewX, viewY, r2, cellMinX, cellMinY, cellMaxX, cellMaxY, epsilon) {
				covered[cellKey{cx: cx, cy: cy}] = true
			}
		}
	}
	count := 0
	for _, object := range objects {
		if covered[struct {
			cx int
			cy int
		}{int(math.Floor(object.X / CellSize)), int(math.Floor(object.Y / CellSize))}] {
			count++
		}
	}
	return count
}

func TestCircleBoundary(t *testing.T) {
	// 方形范围的角落必须排除：8*8+8*8=128 > 100。
	culler := NewCuller([]Object{{ID: "corner", X: 8, Y: 8}})
	if visible := culler.Visible(0, 0, 10); len(visible) != 0 {
		t.Fatalf("圆外角点不该可见：%v", visible)
	}

	// 圆边界上的点必须可见（<=，含等号）。
	onEdge := []Object{
		{ID: "p34", X: 3, Y: 4},  // d2 = 25
		{ID: "axis", X: 5, Y: 0}, // d2 = 25
		{ID: "p68", X: -6, Y: 8}, // d2 = 100
		{ID: "inside", X: 1, Y: 1},
	}
	culler = NewCuller(onEdge)
	got := culler.Visible(0, 0, 5)
	want := []string{"inside", "axis", "p34"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("边界含等号、圆外角点排除：got=%v want=%v", got, want)
	}
}

func TestResultOrderingByDistanceThenID(t *testing.T) {
	objects := []Object{
		{ID: "far", X: 3, Y: 0},
		{ID: "near", X: 1, Y: 0},
		{ID: "mid", X: 2, Y: 0},
	}
	culler := NewCuller(objects)
	got := culler.Visible(0, 0, 10)
	want := []string{"near", "mid", "far"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("应按距离升序返回：got=%v want=%v", got, want)
	}

	// 同距按 ID 升序。
	tie := []Object{
		{ID: "c", X: 0, Y: 2},
		{ID: "a", X: 2, Y: 0},
		{ID: "b", X: -2, Y: 0},
	}
	culler = NewCuller(tie)
	got = culler.Visible(0, 0, 10)
	want = []string{"a", "b", "c"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("同距应按 ID 升序：got=%v want=%v", got, want)
	}
}

func TestDuplicateIDAppearsOnce(t *testing.T) {
	culler := NewCuller([]Object{
		{ID: "boss", X: 1, Y: 0},
		{ID: "boss", X: 1, Y: 0},
	})
	got := culler.Visible(0, 0, 10)
	if len(got) != 1 || got[0] != "boss" {
		t.Fatalf("同一 ID 只应出现一次：%v", got)
	}

	// 同 ID 不同位置时取最近的一个参与排序。
	culler = NewCuller([]Object{
		{ID: "boss", X: 8, Y: 0},
		{ID: "boss", X: 1, Y: 0},
		{ID: "minion", X: 3, Y: 0},
	})
	got = culler.Visible(0, 0, 10)
	want := []string{"boss", "minion"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("重复 ID 应取最近位置：got=%v want=%v", got, want)
	}
}

func TestRepeatedQueriesAreIdentical(t *testing.T) {
	objects := []Object{
		{ID: "a", X: 1, Y: 1},
		{ID: "a", X: 2, Y: 2},
		{ID: "b", X: 0, Y: 3},
		{ID: "c", X: 9, Y: 9},
	}
	culler := NewCuller(objects)
	first := culler.Visible(0, 0, 5)
	for attempt := 0; attempt < 10; attempt++ {
		again := culler.Visible(0, 0, 5)
		if !reflect.DeepEqual(again, first) {
			t.Fatalf("重复查询结果必须相同：%v != %v", again, first)
		}
	}
}

func TestScannedBoundedByCoveredCellsAndHits(t *testing.T) {
	// 5 万对象堆在远处，半径 10 的查询一个都不该扫到。
	objects := make([]Object, 0, 50000)
	for index := 0; index < 50000; index++ {
		objects = append(objects, Object{ID: "far", X: 10000, Y: 10000})
	}
	culler := NewCuller(objects)
	before := culler.Scanned()
	culler.Visible(0, 0, 10)
	if scanned := culler.Scanned() - before; scanned != 0 {
		t.Fatalf("远处对象不该被扫描，实际扫了 %d", scanned)
	}

	// 通用口径：单次扫描数 == 覆盖格里的对象数，因此 <= 覆盖格对象数 + 命中数。
	rng := rand.New(rand.NewSource(42))
	mixed := make([]Object, 0, 2000)
	for index := 0; index < 2000; index++ {
		mixed = append(mixed, Object{
			ID: string(rune('a'+index%7)) + "-" + string(rune('A'+index%3)),
			X:  rng.Float64()*100 - 50,
			Y:  rng.Float64()*100 - 50,
		})
	}
	culler = NewCuller(mixed)
	for query := 0; query < 100; query++ {
		viewX := rng.Float64()*100 - 50
		viewY := rng.Float64()*100 - 50
		radius := rng.Float64() * 20
		before = culler.Scanned()
		hits := culler.Visible(viewX, viewY, radius)
		scanned := culler.Scanned() - before
		covered := coveredObjectCount(mixed, viewX, viewY, radius)
		if scanned > covered+len(hits) {
			t.Fatalf("查询 #%d：scanned=%d 超过覆盖格对象数+命中数=%d+%d", query, scanned, covered, len(hits))
		}
		if scanned != covered {
			t.Fatalf("查询 #%d：应恰好扫描覆盖格中的对象：scanned=%d covered=%d", query, scanned, covered)
		}
	}
}

func TestRandom200MatchesBruteForce(t *testing.T) {
	rng := rand.New(rand.NewSource(20240930))
	for trial := 0; trial < 200; trial++ {
		count := rng.Intn(300) + 1
		objects := make([]Object, count)
		duplicates := rng.Intn(5)
		for index := range objects {
			objects[index] = Object{
				ID: "id-" + string(rune('a'+rng.Intn(20))) + string(rune('a'+rng.Intn(20))) + "-" + itoa(rng.Intn(count+duplicates)),
				X:  rng.NormFloat64() * 15,
				Y:  rng.NormFloat64() * 15,
			}
		}
		// 刻意制造一些重复 ID。
		for dup := 0; dup < duplicates; dup++ {
			src := rng.Intn(count)
			dst := rng.Intn(count)
			objects[dst].ID = objects[src].ID
		}
		viewX := rng.NormFloat64() * 10
		viewY := rng.NormFloat64() * 10
		radius := rng.Float64() * 25

		culler := NewCuller(objects)
		got := culler.Visible(viewX, viewY, radius)
		want := bruteVisible(objects, viewX, viewY, radius)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("第 %d 组随机数据与参考实现不一致：\n got=%v\nwant=%v", trial, got, want)
		}
		// 扫描代价口径。
		covered := coveredObjectCount(objects, viewX, viewY, radius)
		if culler.Scanned() > covered+len(want) {
			t.Fatalf("第 %d 组：scanned=%d 超过 %d+%d", trial, culler.Scanned(), covered, len(want))
		}
	}
}

func TestLargeScale200K(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	const n = 200000
	objects := make([]Object, n)
	for index := range objects {
		objects[index] = Object{
			ID: "obj-" + itoa(index),
			X:  rng.Float64() * 2000,
			Y:  rng.Float64() * 2000,
		}
	}
	culler := NewCuller(objects)
	viewX, viewY, radius := 1000.0, 1000.0, 12.0
	before := culler.Scanned()
	got := culler.Visible(viewX, viewY, radius)
	scanned := culler.Scanned() - before
	want := bruteVisible(objects, viewX, viewY, radius)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("20 万规模结果与参考实现不一致：len(got)=%d len(want)=%d", len(got), len(want))
	}
	covered := coveredObjectCount(objects, viewX, viewY, radius)
	if scanned > covered+len(want) {
		t.Fatalf("20 万规模扫描越界：scanned=%d covered=%d hits=%d", scanned, covered, len(want))
	}
	if scanned >= n/2 {
		t.Fatalf("查询退化为全量扫描：scanned=%d / %d", scanned, n)
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}

func BenchmarkVisible200K(b *testing.B) {
	rng := rand.New(rand.NewSource(11))
	const n = 200000
	objects := make([]Object, n)
	for index := range objects {
		objects[index] = Object{
			ID: "obj-" + itoa(index),
			X:  rng.Float64() * 2000,
			Y:  rng.Float64() * 2000,
		}
	}
	culler := NewCuller(objects)
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		culler.Visible(1000, 1000, 12)
	}
}
