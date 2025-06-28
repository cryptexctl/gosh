//go:build !noeaster
// +build !noeaster

package shell

import "gosh/internal/builtin"

func registerEaster(b *builtin.Manager) {
	b.Register("gosha", func(args []string) int {
		println("Это не смешно!")
		return 0
	})

	b.Register("bash", func(args []string) int {
		println("Bash is too old.")
		return 0
	})
}
