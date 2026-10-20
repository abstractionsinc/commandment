//go:build windows

package shelltest

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

type fileAttributeTagInfo struct {
	attributes uint32
	reparseTag uint32
}

type fileDispositionInfo struct {
	deleteFile byte
}

type fileCaseSensitiveInfo struct {
	flags uint32
}

func prepareOperationWorkspace(workspace string) error {
	root, err := openWorkspaceRoot(workspace)
	if err != nil {
		return failureError{
			subject: "open operation workspace",
			cause:   err,
		}
	}
	defer func() { _ = windows.CloseHandle(root) }()
	if err := rejectWorkspaceReparse(root); err != nil {
		return err
	}
	return nil
}

func createWorkspaceFile(workspace, relative string, content []byte, _ os.FileMode) error {
	return withWorkspaceParent(workspace, relative, func(parent windows.Handle, name string) error {
		handle, err := createWorkspaceChild(parent, name, false)
		if err != nil {
			return failureError{
				subject: "create workspace file",
				detail:  relative,
				cause:   err,
			}
		}
		file := os.NewFile(uintptr(handle), name)
		if file == nil {
			_ = windows.CloseHandle(handle)
			return failureError{
				subject: "create workspace file handle",
				detail:  relative,
			}
		}
		remove := true
		defer func() {
			if remove {
				_ = markWorkspacePathForDeletion(handle)
			}
			_ = file.Close()
		}()
		if _, err := file.Write(content); err != nil {
			return failureError{
				subject: "write workspace file",
				detail:  relative,
				cause:   err,
			}
		}
		if err := file.Sync(); err != nil {
			return failureError{
				subject: "flush workspace file",
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
	})
}

func copyWorkspaceExecutable(workspace, relative, source string) error {
	input, err := os.Open(source) // #nosec G304 -- source is os.Executable(), not caller-authored workspace input.
	if err != nil {
		return failureError{
			subject: "open helper executable source",
			cause:   err,
		}
	}
	defer func() { _ = input.Close() }()

	return withWorkspaceParent(workspace, relative, func(parent windows.Handle, name string) error {
		handle, err := createWorkspaceChild(parent, name, false)
		if err != nil {
			return failureError{
				subject: "create helper executable",
				detail:  relative,
				cause:   err,
			}
		}
		output := os.NewFile(uintptr(handle), name)
		if output == nil {
			_ = windows.CloseHandle(handle)
			return failureError{
				subject: "create helper executable handle",
				detail:  relative,
			}
		}
		remove := true
		defer func() {
			if remove {
				_ = markWorkspacePathForDeletion(handle)
			}
			_ = output.Close()
		}()
		if _, err := io.Copy(output, input); err != nil {
			return failureError{
				subject: "copy helper executable",
				detail:  relative,
				cause:   err,
			}
		}
		if err := output.Sync(); err != nil {
			return failureError{
				subject: "flush helper executable",
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
	})
}

func removeWorkspaceFile(workspace, relative string) error {
	return removeWorkspacePath(workspace, relative, false)
}

func removeWorkspaceDirectory(workspace, relative string) error {
	return removeWorkspacePath(workspace, relative, true)
}

func removeWorkspacePath(workspace, relative string, directory bool) error {
	return withWorkspaceParent(workspace, relative, func(parent windows.Handle, name string) error {
		handle, err := openWorkspaceChild(parent, name, directory)
		if err != nil {
			return failureError{
				subject: "open workspace path for removal",
				detail:  relative,
				cause:   err,
			}
		}
		defer func() { _ = windows.CloseHandle(handle) }()
		if err := markWorkspacePathForDeletion(handle); err != nil {
			return failureError{
				subject: "remove workspace path",
				detail:  relative,
				cause:   err,
			}
		}
		return nil
	})
}

func withWorkspaceParent(workspace, relative string, operation func(windows.Handle, string) error) error {
	root, err := openWorkspaceRoot(workspace)
	if err != nil {
		return failureError{
			subject: "open operation workspace",
			cause:   err,
		}
	}
	defer func() { _ = windows.CloseHandle(root) }()
	if err := rejectWorkspaceReparse(root); err != nil {
		return err
	}

	parts := strings.Split(filepath.Clean(relative), string(filepath.Separator))
	if len(parts) == 0 || parts[len(parts)-1] == "" {
		return failureError{subject: "invalid empty workspace path"}
	}
	parent := root
	owned := make([]windows.Handle, 0, len(parts)-1)
	defer func() {
		for index := len(owned) - 1; index >= 0; index-- {
			_ = windows.CloseHandle(owned[index])
		}
	}()

	for _, component := range parts[:len(parts)-1] {
		directory, openErr := openWorkspaceChild(parent, component, true)
		if openErr != nil {
			if !workspaceObjectMissing(openErr) {
				return failureError{
					subject: "open workspace directory",
					detail:  component,
					cause:   openErr,
				}
			}
			directory, openErr = createWorkspaceChild(parent, component, true)
			if openErr != nil {
				return failureError{
					subject: "create workspace directory",
					detail:  component,
					cause:   openErr,
				}
			}
		}
		if err := rejectWorkspaceReparse(directory); err != nil {
			_ = windows.CloseHandle(directory)
			return err
		}
		owned = append(owned, directory)
		parent = directory
	}

	return operation(parent, parts[len(parts)-1])
}

func openWorkspaceRoot(workspace string) (windows.Handle, error) {
	path := filepath.Clean(workspace)
	if strings.HasPrefix(path, `\\`) {
		path = `\??\UNC\` + strings.TrimPrefix(path, `\\`)
	} else {
		path = `\??\` + path
	}
	name, err := windows.NewNTUnicodeString(path)
	if err != nil {
		return 0, err
	}
	attributes, err := workspaceObjectAttributes(0, name)
	if err != nil {
		return 0, err
	}
	return ntOpenWorkspaceObject(&attributes, true, windows.FILE_OPEN)
}

func openWorkspaceChild(parent windows.Handle, name string, directory bool) (windows.Handle, error) {
	objectName, err := windows.NewNTUnicodeString(name)
	if err != nil {
		return 0, err
	}
	attributes, err := workspaceObjectAttributes(parent, objectName)
	if err != nil {
		return 0, err
	}
	return ntOpenWorkspaceObject(&attributes, directory, windows.FILE_OPEN)
}

func createWorkspaceChild(parent windows.Handle, name string, directory bool) (windows.Handle, error) {
	objectName, err := windows.NewNTUnicodeString(name)
	if err != nil {
		return 0, err
	}
	attributes, err := workspaceObjectAttributes(parent, objectName)
	if err != nil {
		return 0, err
	}
	handle, err := ntOpenWorkspaceObject(&attributes, directory, windows.FILE_CREATE)
	if err != nil {
		return 0, err
	}
	if err := rejectWorkspaceReparse(handle); err != nil {
		_ = windows.CloseHandle(handle)
		return 0, err
	}
	return handle, nil
}

func workspaceObjectAttributes(parent windows.Handle, name *windows.NTUnicodeString) (windows.OBJECT_ATTRIBUTES, error) {
	flags := uint32(windows.OBJ_DONT_REPARSE)
	if parent == 0 {
		flags |= windows.OBJ_CASE_INSENSITIVE
	} else {
		caseSensitive, err := workspaceDirectoryCaseSensitive(parent)
		if err != nil {
			return windows.OBJECT_ATTRIBUTES{}, err
		}
		if !caseSensitive {
			flags |= windows.OBJ_CASE_INSENSITIVE
		}
	}
	attributes := windows.OBJECT_ATTRIBUTES{
		Length:        uint32(unsafe.Sizeof(windows.OBJECT_ATTRIBUTES{})),
		RootDirectory: parent,
		ObjectName:    name,
		Attributes:    flags,
	}
	return attributes, nil
}

func workspaceDirectoryCaseSensitive(handle windows.Handle) (bool, error) {
	var info fileCaseSensitiveInfo
	if err := windows.GetFileInformationByHandleEx(handle, windows.FileCaseSensitiveInfo, (*byte)(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info))); err != nil {
		return false, failureError{
			subject: "inspect workspace directory case semantics",
			cause:   err,
		}
	}
	return info.flags&windows.FILE_CS_FLAG_CASE_SENSITIVE_DIR != 0, nil
}

func ntOpenWorkspaceObject(attributes *windows.OBJECT_ATTRIBUTES, directory bool, disposition uint32) (windows.Handle, error) {
	options := uint32(windows.FILE_OPEN_REPARSE_POINT | windows.FILE_SYNCHRONOUS_IO_NONALERT)
	fileAttributes := uint32(windows.FILE_ATTRIBUTE_NORMAL)
	share := uint32(0)
	if directory {
		options |= windows.FILE_DIRECTORY_FILE
		fileAttributes = windows.FILE_ATTRIBUTE_DIRECTORY
		share = windows.FILE_SHARE_READ | windows.FILE_SHARE_WRITE | windows.FILE_SHARE_DELETE
	} else {
		options |= windows.FILE_NON_DIRECTORY_FILE
	}
	access := uint32(windows.FILE_GENERIC_READ | windows.FILE_GENERIC_WRITE | windows.DELETE)
	var status windows.IO_STATUS_BLOCK
	var handle windows.Handle
	var allocation int64
	err := windows.NtCreateFile(&handle, access, attributes, &status, &allocation, fileAttributes, share, disposition, options, 0, 0)
	if err != nil {
		return 0, err
	}
	return handle, nil
}

func rejectWorkspaceReparse(handle windows.Handle) error {
	var info fileAttributeTagInfo
	if err := windows.GetFileInformationByHandleEx(handle, windows.FileAttributeTagInfo, (*byte)(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info))); err != nil {
		return failureError{
			subject: "inspect workspace filesystem object",
			cause:   err,
		}
	}
	if info.attributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
		return failureError{subject: "workspace path contains a reparse point"}
	}
	return nil
}

func workspaceObjectMissing(err error) bool {
	status, ok := err.(windows.NTStatus)
	if !ok {
		return false
	}
	return status == windows.STATUS_OBJECT_NAME_NOT_FOUND || status == windows.STATUS_OBJECT_PATH_NOT_FOUND
}

func markWorkspacePathForDeletion(handle windows.Handle) error {
	info := fileDispositionInfo{deleteFile: 1}
	return windows.SetFileInformationByHandle(handle, windows.FileDispositionInfo, (*byte)(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info)))
}
