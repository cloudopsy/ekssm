package util

import (
	"fmt"
	"io"
	"os"

	"github.com/cloudopsy/ekssm/internal/logging"
)

func CopyFile(src, dst string) error {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		if os.IsNotExist(err) {
			// If source doesn't exist, there's nothing to backup.
			logging.Debugf("Source file %s does not exist, skipping backup.", src)
			return nil // Not an error, just nothing to copy
		}
		return fmt.Errorf("failed to stat source file %s: %w", src, err)
	}

	if !sourceFileStat.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %w", src, err)
	}
	defer source.Close()

	destination, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, sourceFileStat.Mode())
	if err != nil {
		return fmt.Errorf("failed to create destination file %s: %w", dst, err)
	}
	defer destination.Close()

	nBytes, err := io.Copy(destination, source)
	if err != nil {
		// Attempt to remove partially written destination file on error
		_ = os.Remove(dst)
		return fmt.Errorf("failed to copy file contents from %s to %s: %w", src, dst, err)
	}
	logging.Debugf("Copied %d bytes from %s to %s", nBytes, src, dst)

	return nil
}
