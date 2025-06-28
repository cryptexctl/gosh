//go:build !noeaster
// +build !noeaster

package shell

import (
	"fmt"
	"gosh/internal/builtin"
	"os"
)

func registerEaster(b *builtin.Manager) {
	b.Register("gosha", func(args []string) int {
		path, _ := os.Executable()
		fmt.Printf("Это не смешно! (%s)\n", path)
		return 0
	})

	b.Register("bash", func(args []string) int {
		path, _ := os.Executable()
		fmt.Printf("Bash is too old. Try %s\n", path)
		return 0
	})

	b.Register("oh my", func(args []string) int {
		path, _ := os.Executable()
		fmt.Printf("gosh! (%s)\n", path)
		return 0
	})
}
