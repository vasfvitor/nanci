package main

// dialogCanceledMessage is the error text Wails v3 returns on Windows when the
// user closes a file dialog without picking anything. The error value lives in
// an internal Wails package (internal/go-common-file-dialog/cfd), so only its
// text can be matched.
const dialogCanceledMessage = "cancelled by user"

// dialogPath turns a closed file dialog into an empty path, as Wails v2 did,
// so the frontend keeps treating "" as "nothing picked".
func dialogPath(path string, err error) (string, error) {
	if err != nil && err.Error() == dialogCanceledMessage {
		return "", nil
	}
	return path, err
}

// dialogPaths does the same for a multiple selection: a closed dialog is no
// paths and no error.
func dialogPaths(paths []string, err error) ([]string, error) {
	if err != nil && err.Error() == dialogCanceledMessage {
		return nil, nil
	}
	return paths, err
}
