package flexvol

import "fmt"

type Status string

const (
	StatusSuccess      Status = "Success"
	StatusFailure      Status = "Failure"
	StatusNotSupported Status = "Not supported"
)

func Unsupported(call string) DriverStatus {
	return DriverStatus{
		Status:  StatusNotSupported,
		Message: fmt.Sprintf("%s not supported", call),
	}
}
