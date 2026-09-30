# scene-cull

场景剔除：按空间网格与可视半径剔除，结果不漏且有序，只用 Go 标准库。

```
go test ./...
go run ./cmd/scene-cull --sample corner
go run ./cmd/scene-cull --sample order
go run ./cmd/scene-cull --sample dup
```

## 口径（README 为准）

- **圆形可视范围**：只有 `dx*dx + dy*dy <= radius*radius` 的对象才可见（方形范围会把角落误判进来）。
- **结果有序**：按距离升序返回，同距按 ID 升序；同一 ID 只出现一次。
- **按网格扫描**：用 `CellSize` 网格只扫可视范围覆盖到的格子，
  `Scanned` 不超过覆盖格里的对象数加命中数，不许每次全量扫描场景。
- **不变量**：剔除结果与「全量两两判定」的暴力实现一致（不漏不多）；重复查询结果相同。
- **规模**：单场景 20 万对象、每秒 10 万次查询，单次 O(覆盖格 + 命中数)，内存 O(对象数)。

## 输出契约（不改格式）

```
visible=0
order=near,mid,far
count=1
scanned<=6000
```

## 目录

```
cull.go                 网格剔除
cmd/scene-cull          命令行入口
cull_test.go            go test 用例
```
