package astios

import (
	"context"
	"os"

	"github.com/asticode/go-astitools/io"
)

// Copy is a cross partitions cancellable copy
func Copy(ctx context.Context, src, dst string) (err error) {
	// Check context
	if err = ctx.Err(); err != nil {
		return
	}

	// Open the source file
	srcFile, err := os.Open(src)
	if err != nil {
		return
	}
	defer srcFile.Close()

	// Check context
	if err = ctx.Err(); err != nil {
		return
	}

	// Create the destination file
	dstFile, err := os.Create(dst)
	if err != nil {
		return
	}
	defer dstFile.Close()

	// Check context
	if err = ctx.Err(); err != nil {
		return
	}

	// Copy the content
	_, err = astiio.Copy(ctx, srcFile, dstFile)
	return
}
