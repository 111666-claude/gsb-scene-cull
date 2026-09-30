package scenecull

import "testing"

func TestInsideRadiusIsVisible(t *testing.T) {
	culler := NewCuller([]Object{{ID: "near", X: 1, Y: 1}})
	visible := culler.Visible(0, 0, 5)
	if len(visible) != 1 || visible[0] != "near" {
		t.Fatalf("半径内的对象应该可见：%v", visible)
	}
}

func TestFarObjectIsHidden(t *testing.T) {
	culler := NewCuller([]Object{{ID: "far", X: 100, Y: 0}})
	if visible := culler.Visible(0, 0, 5); len(visible) != 0 {
		t.Fatalf("半径外的对象不该可见：%v", visible)
	}
}

func TestScannedStartsAtZero(t *testing.T) {
	if NewCuller(nil).Scanned() != 0 {
		t.Fatal("初始扫描数应该是 0")
	}
}

func TestConstants(t *testing.T) {
	if CellSize != 10 {
		t.Fatal("网格常量被改了")
	}
}
