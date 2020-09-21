/*
Copyright © 2020 Her Majesty the Queen in Right of Canada, as represented by the Minister of Statistics Canada

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strconv"
	"syscall"

	"github.com/StatCan/boathouse/internal/flexvol"
	"github.com/StatCan/boathouse/internal/utils"
	"github.com/spf13/cobra"
)

// unmountCmd represents the unmount command
var unmountCmd = &cobra.Command{
	Use:   "unmount",
	Short: "Unmount a volume from the mount directory",
	Run: func(cmd *cobra.Command, args []string) {
		target := args[0]

		pidfile := path.Join(os.TempDir(), fmt.Sprintf("%s.pid", utils.PathSum256(target)))
		pidstr, err := ioutil.ReadFile(pidfile)
		if err != nil {
			perr := utils.PrintJSON(os.Stdout, flexvol.DriverStatus{
				Status:  flexvol.StatusFailure,
				Message: fmt.Sprintf("could not load pid file for path %s: %v", target, err),
			})
			if perr != nil {
				log.Fatal(perr)
			}
			os.Exit(0)
		}

		// 1. Find pid for mount and terminate
		pid, err := strconv.Atoi(string(pidstr))
		if err != nil {
			perr := utils.PrintJSON(os.Stdout, flexvol.DriverStatus{
				Status:  flexvol.StatusFailure,
				Message: fmt.Sprintf("error reading pid at %s: %v", target, err),
			})
			if perr != nil {
				log.Fatal(perr)
			}
			os.Exit(0)
		}

		proc, err := os.FindProcess(pid)
		if err != nil {
			perr := utils.PrintJSON(os.Stdout, flexvol.DriverStatus{
				Status:  flexvol.StatusFailure,
				Message: fmt.Sprintf("cannot find process %d: %v", pid, err),
			})
			if perr != nil {
				log.Fatal(perr)
			}
			os.Exit(0)
		}
		err = proc.Signal(syscall.SIGTERM)
		if err != nil && err.Error() != "os: process already finished" {
			perr := utils.PrintJSON(os.Stdout, flexvol.DriverStatus{
				Status:  flexvol.StatusFailure,
				Message: fmt.Sprintf("error sending signal to pid %d: %v", pid, err),
			})
			if perr != nil {
				log.Fatal(perr)
			}
			os.Exit(0)
		}

		err = os.Remove(target)
		if err != nil && !os.IsNotExist(err) {
			perr := utils.PrintJSON(os.Stdout, flexvol.DriverStatus{
				Status:  flexvol.StatusFailure,
				Message: fmt.Sprintf("error removing target %s: %v", target, err),
			})
			if perr != nil {
				log.Fatal(perr)
			}
			os.Exit(0)
		}

		// Remove the pid file
		_ = os.Remove(pidfile)

		perr := utils.PrintJSON(os.Stdout, flexvol.DriverStatus{
			Status:  flexvol.StatusSuccess,
			Message: "terminated",
		})
		if perr != nil {
			log.Fatal(perr)
		}
	},
}

func init() {
	rootCmd.AddCommand(unmountCmd)
}
