// Command scene-cull 跑场景剔除样例。
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"example.com/scenecull"
)

// Run 执行一次命令行调用，返回退出码。
func Run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("scene-cull", flag.ContinueOnError)
	flags.SetOutput(stderr)
	sample := flags.String("sample", "corner", "corner / order / dup / scan")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	switch *sample {
	case "corner":
		culler := scenecull.NewCuller([]scenecull.Object{{ID: "corner", X: 8, Y: 8}})
		fmt.Fprintf(stdout, "visible=%d\n", len(culler.Visible(0, 0, 10)))
	case "order":
		culler := scenecull.NewCuller([]scenecull.Object{
			{ID: "far", X: 3, Y: 0},
			{ID: "near", X: 1, Y: 0},
			{ID: "mid", X: 2, Y: 0},
		})
		fmt.Fprintf(stdout, "order=%s\n", strings.Join(culler.Visible(0, 0, 10), ","))
	case "dup":
		culler := scenecull.NewCuller([]scenecull.Object{
			{ID: "boss", X: 1, Y: 0},
			{ID: "boss", X: 1, Y: 0},
		})
		fmt.Fprintf(stdout, "count=%d\n", len(culler.Visible(0, 0, 10)))
	case "scan":
		objects := make([]scenecull.Object, 0, 50000)
		for index := 0; index < 50000; index++ {
			objects = append(objects, scenecull.Object{ID: fmt.Sprintf("o-%d", index), X: 10000, Y: 10000})
		}
		culler := scenecull.NewCuller(objects)
		for query := 0; query < 1000; query++ {
			culler.Visible(0, 0, 10)
		}
		fmt.Fprintf(stdout, "scanned=%d\n", culler.Scanned())
	default:
		fmt.Fprintln(stderr, "需要 --sample corner|order|dup|scan")
		return 2
	}
	return 0
}

func main() {
	os.Exit(Run(os.Args[1:], os.Stdout, os.Stderr))
}
