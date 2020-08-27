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
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/StatCan/boathouse/internal/agent"
	"github.com/StatCan/boathouse/internal/client"
	"github.com/spf13/cobra"
	"k8s.io/klog"
)

// mountCmd represents the mount command
var mountCmd = &cobra.Command{
	Use:   "mount",
	Short: "Mount a volume to a directory",
	// Args:      cobra.ExactValidArgs(2),
	// ValidArgs: []string{"mountdir", "options"},
	Run: func(cmd *cobra.Command, args []string) {

		// 0. Decode options
		var options map[string]string
		err := json.Unmarshal([]byte(args[1]), &options)
		if err != nil {
			klog.Fatalf("failed to parse options: %v", err)
		}

		// 1. @TODO: Fork ourselves (we want to run in the background)
		socketPath, _ := net.ResolveUnixAddr("unix", "/tmp/boathouse.sock")
		c, err := client.NewClient(socketPath)
		if err != nil {
			klog.Fatalf("failed to create client: %v")
		}

		// 2. Request credentials from the agent
		creds, err := c.IssueCredentials(agent.IssueCredentialRequest{
			Path: "minio_minimal_tenant1/keys/profile-zachary-seguin",
		})
		if err != nil {
			klog.Fatalf("failed to issue credentials: %v", err)
		}
		log.Printf("%v", creds)

		// 3. [Client] Start goofys

		// TODO: Setup a context that will cancel when the process is requested to terminate
		ctx := context.Background()

		goofysArgs := []string{}

		// Run in foreground mode
		goofysArgs = append(goofysArgs, "-f")

		// File/Directory modes
		dirMode := "0755"
		if val, ok := options["dirMode"]; ok {
			dirMode = val
		}

		fileMode := "0644"
		if val, ok := options["fileMode"]; ok {
			fileMode = val
		}

		goofysArgs = append(goofysArgs,
			"-o", "allow_other",
			"--dir-mode", dirMode,
			"--file-mode", fileMode,
		)

		// Endpoint
		if val, ok := options["endpoint"]; ok {
			goofysArgs = append(goofysArgs, "--endpoint", val)
		}

		// Region
		if val, ok := options["region"]; ok {
			goofysArgs = append(goofysArgs, "--region", val)
		}

		// UID
		if val, ok := options["uid"]; ok {
			goofysArgs = append(goofysArgs, "--uid", val)
		}

		// GID
		if val, ok := options["gid"]; ok {
			goofysArgs = append(goofysArgs, "--gid", val)
		}

		// Debug
		if val, ok := options["debug_s3"]; ok {
			bval, err := strconv.ParseBool(val)
			if err != nil {
				klog.Warningf("failed to parse bool for debug_s3: %s : %v", val, err)
			}
			if bval {
				goofysArgs = append(goofysArgs, "--debug_s3")
			}
		}

		// Bucket (positional argument)
		if val, ok := options["bucket"]; ok {
			goofysArgs = append(goofysArgs, val)
		} else {
			klog.Fatalf("bucket option is required")
		}

		// Mount path (positional argument)
		goofysArgs = append(goofysArgs, args[0])

		// Setup a new context, with the existing context as a parent,
		// which will automatically terminate goofys when our
		// credentials expire
		credscontext, _ := context.WithDeadline(ctx, time.Now().Add(creds.Lease.Duration))

		goofys := exec.CommandContext(credscontext, "goofys", goofysArgs...)

		// Setup environment variable for access key and secret key
		goofys.Env = os.Environ()
		goofys.Env = append(goofys.Env, fmt.Sprintf("AWS_ACCESS_KEY_ID=%s", creds.AccessKey))
		goofys.Env = append(goofys.Env, fmt.Sprintf("AWS_SECRET_ACCESS_KEY=%s", creds.SecretKey))

		err = goofys.Run()

		if err != nil {
			klog.Fatalf("failed running goofys: %v", err)
		}

		// 4a. [Client] On success, signal parent that we have successfully started and write pid to state file
		// 4ai: [Client] Timeout until credentials expire. On expiry, request new credentials, terminate goofys and start goofys
		// 4b. [Client] On failure, signal parent that we have not started. Exit.
		// 5. [Parent] Exits and returns success/failure based on signal

		log.Fatal("mount called: not implemented")
	},
}

func init() {
	rootCmd.AddCommand(mountCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// mountCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// mountCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
