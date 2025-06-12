package linux

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"
	"time"
)

type FileSystem struct{}

// NewFileSystem creates a new instance of FileSystem
func NewFileSystem() *FileSystem {
	return &FileSystem{}
}

// SetCreationTime changes the file creation time by temporarily modifying the system time
func (fs *FileSystem) SetCreationTime(filePath string, newCrtime time.Time) error {
	stat, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("stat original: %w", err)
	}
	mtime := stat.ModTime()

	var atime time.Time
	var uid, gid int
	var mode os.FileMode

	if sysStat, ok := stat.Sys().(*syscall.Stat_t); ok {
		atime = time.Unix(sysStat.Atim.Sec, sysStat.Atim.Nsec)
		uid = int(sysStat.Uid)
		gid = int(sysStat.Gid)
		mode = stat.Mode()
	} else {
		return fmt.Errorf("failed to retrieve file attributes")
	}

	// Temporary file — rename the original
	tmpPath := filePath + ".bak_for_crtime"
	if err := os.Rename(filePath, tmpPath); err != nil {
		return fmt.Errorf("rename: %w", err)
	}

	// Set the system time
	fmt.Println(">> Changing system time to:", newCrtime)
	if err := setSystemTime(newCrtime); err != nil {
		return fmt.Errorf("set system time: %w", err)
	}

	// Copy the temporary file back to the original
	if err := copyFile(tmpPath, filePath); err != nil {
		return fmt.Errorf("copy back: %w", err)
	}

	// Remove the temporary file
	_ = os.Remove(tmpPath)

	// Restore the system time
	fmt.Println(">> Restoring system time")
	if err := restoreSystemTime(); err != nil {
		return fmt.Errorf("restore system time: %w", err)
	}

	// Restore owner, permissions, and atime/mtime
	if err := os.Chown(filePath, uid, gid); err != nil {
		return fmt.Errorf("chown: %w", err)
	}
	if err := os.Chmod(filePath, mode); err != nil {
		return fmt.Errorf("chmod: %w", err)
	}
	timespec := []syscall.Timespec{
		syscall.NsecToTimespec(atime.UnixNano()),
		syscall.NsecToTimespec(mtime.UnixNano()),
	}
	if err := syscall.UtimesNano(filePath, timespec); err != nil {
		return fmt.Errorf("restore atime/mtime: %w", err)
	}

	return nil
}

func setSystemTime(t time.Time) error {
	formatted := t.Format("2006-01-02 15:04:05")
	cmd := exec.Command("date", "-s", formatted)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func restoreSystemTime() error {
	cmd := exec.Command("date", "-s", "now")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func copyFile(src, dst string) error {
	input, err := os.Open(src)
	if err != nil {
		return err
	}
	defer input.Close()

	output, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer output.Close()

	_, err = io.Copy(output, input)
	return err
}
