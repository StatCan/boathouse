package utils

import (
	"encoding/json"
	"fmt"
	"io"
)

// PrintJSON prints obj as JSON to w.
func PrintJSON(w io.Writer, obj interface{}) error {
	b, err := json.Marshal(obj)
	if err != nil {
		return err
	}

	_, err = fmt.Fprint(w, string(b))
	return err
}
