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
		fmt.Printf("Это не смешно!\n")
		return 0
	})

	b.Register("bash", func(args []string) int {
		fmt.Printf("Bash is too old.\n")
		return 0
	})

	b.Register("ohmy", func(args []string) int {
		path, _ := os.Executable()
		fmt.Printf("%s\n", path)
		return 0
	})
}
