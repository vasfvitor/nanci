package main

// dialogCanceledMessage is the error text Wails v3 returns on Windows when the
// user closes a file dialog without picking anything. The error value lives in
// an internal Wails package (internal/go-common-file-dialog/cfd), so only its
// text can be matched.
const dialogCanceledMessage = "cancelled by user"

// dialogResult turns a closed file dialog into the zero value and no error:
// an empty path or no paths, which the frontend treats as "nothing picked".
func dialogResult[T any](v T, err error) (T, error) {
	if err != nil && err.Error() == dialogCanceledMessage {
		var zero T
		return zero, nil
	}
	return v, err
}
