package astihttp

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/asticode/go-astitools/io"
	"github.com/pkg/errors"
)

// Download is a cancellable function that downloads a src into a dst using a specific *http.Client
func Download(ctx context.Context, c *http.Client, src, dst string) (err error) {
	// Create the dst file
	var f *os.File
	if f, err = os.Create(dst); err != nil {
		return errors.Wrapf(err, "creating file %s failed", dst)
	}
	defer f.Close()

	// Send request
	var resp *http.Response
	if resp, err = c.Get(src); err != nil {
		return errors.Wrapf(err, "getting %s failed", src)
	}
	defer resp.Body.Close()

	// Validate status code
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("getting %s returned %d status code", src, resp.StatusCode)
	}

	// Copy
	if _, err = astiio.Copy(ctx, resp.Body, f); err != nil {
		return errors.Wrapf(err, "copying content from %s to %s failed", src, dst)
	}
	return
}
