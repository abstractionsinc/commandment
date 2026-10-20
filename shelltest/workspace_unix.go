//go:build !windows

package shelltest

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"golang.org/x/sys/unix"
)

const (
	workspaceOpenSubject = "open operation workspace"
)

func prepareOperationWorkspace(workspace string) error {
	if err := os.Chmod(workspace, privateDirectoryMode); err != nil {
		return failureError{
			subject: "make operation workspace private",
			cause:   err,
		}
	}
	return nil
}

func createWorkspaceFile(workspace, relative string, content []byte, mode os.FileMode) error {
	root, err := unix.Open(workspace, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return failureError{
			subject: workspaceOpenSubject,
			cause:   err,
		}
	}
	defer func() { _ = unix.Close(root) }()

	parent, leaf, cleanup, err := openWorkspaceParent(root, relative)
	if err != nil {
		return err
	}
	defer cleanup()

	fd, err := unix.Openat(parent, leaf, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_CLOEXEC|unix.O_NOFOLLOW, uint32(mode.Perm()))
	if err != nil {
		return failureError{
			subject: "create workspace file",
			detail:  relative,
			cause:   err,
		}
	}
	file := os.NewFile(uintptr(fd), leaf)
	if file == nil {
		_ = unix.Close(fd)
		return failureError{
			subject: "create workspace file handle",
			detail:  relative,
		}
	}
	remove := true
	defer func() {
		_ = file.Close()
		if remove {
			_ = unix.Unlinkat(parent, leaf, 0)
		}
	}()
	if _, err := file.Write(content); err != nil {
		return failureError{
			subject: "write workspace file",
			detail:  relative,
			cause:   err,
		}
	}
	if err := file.Close(); err != nil {
		return failureError{
			subject: "close workspace file",
			detail:  relative,
			cause:   err,
		}
	}
	remove = false
	return nil
}

func copyWorkspaceExecutable(workspace, relative, source string) error {
	root, err := unix.Open(workspace, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return failureError{
			subject: workspaceOpenSubject,
			cause:   err,
		}
	}
	defer func() { _ = unix.Close(root) }()

	parent, leaf, cleanup, err := openWorkspaceParent(root, relative)
	if err != nil {
		return err
	}
	defer cleanup()

	input, err := os.Open(source) // #nosec G304 -- source is os.Executable(), not caller-authored workspace input.
	if err != nil {
		return failureError{
			subject: "open helper executable source",
			cause:   err,
		}
	}
	defer func() { _ = input.Close() }()

	fd, err := unix.Openat(parent, leaf, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_CLOEXEC|unix.O_NOFOLLOW, privateExecutableMode)
	if err != nil {
		return failureError{
			subject: "create helper executable",
			detail:  relative,
			cause:   err,
		}
	}
	output := os.NewFile(uintptr(fd), leaf)
	if output == nil {
		_ = unix.Close(fd)
		return failureError{
			subject: "create helper executable handle",
			detail:  relative,
		}
	}
	remove := true
	defer func() {
		_ = output.Close()
		if remove {
			_ = unix.Unlinkat(parent, leaf, 0)
		}
	}()
	if _, err := io.Copy(output, input); err != nil {
		return failureError{
			subject: "copy helper executable",
			detail:  relative,
			cause:   err,
		}
	}
	if err := output.Close(); err != nil {
		return failureError{
			subject: "close helper executable",
			detail:  relative,
			cause:   err,
		}
	}
	remove = false
	return nil
}

func removeWorkspaceFile(workspace, relative string) error {
	return removeWorkspacePath(workspace, relative, 0)
}

func removeWorkspaceDirectory(workspace, relative string) error {
	return removeWorkspacePath(workspace, relative, unix.AT_REMOVEDIR)
}

func removeWorkspacePath(workspace, relative string, flags int) error {
	root, err := unix.Open(workspace, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return failureError{
			subject: workspaceOpenSubject,
			cause:   err,
		}
	}
	defer func() { _ = unix.Close(root) }()

	parent, leaf, cleanup, err := openWorkspaceParent(root, relative)
	if err != nil {
		return err
	}
	defer cleanup()
	if err := unix.Unlinkat(parent, leaf, flags); err != nil {
		return failureError{
			subject: "remove workspace path",
			detail:  relative,
			cause:   err,
		}
	}
	return nil
}

func openWorkspaceParent(root int, relative string) (int, string, func(), error) {
	parts := strings.Split(filepath.Clean(relative), string(filepath.Separator))
	if len(parts) == 0 || parts[len(parts)-1] == "" {
		return -1, "", func() {}, failureError{subject: "invalid empty workspace path"}
	}

	parent := root
	opened := make([]int, 0, len(parts)-1)
	cleanup := func() {
		for _, descriptor := range slices.Backward(opened) {
			_ = unix.Close(descriptor)
		}
	}
	for _, component := range parts[:len(parts)-1] {
		descriptor, err := unix.Openat(parent, component, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
		if errors.Is(err, unix.ENOENT) {
			if err := unix.Mkdirat(parent, component, privateDirectoryMode); err != nil {
				cleanup()
				return -1, "", func() {}, failureError{
					subject: "create workspace directory",
					detail:  component,
					cause:   err,
				}
			}
			descriptor, err = unix.Openat(parent, component, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
		}
		if err != nil {
			cleanup()
			return -1, "", func() {}, failureError{
				subject: "open workspace directory",
				detail:  component,
				cause:   err,
			}
		}
		opened = append(opened, descriptor)
		parent = descriptor
	}
	return parent, parts[len(parts)-1], cleanup, nil
}
