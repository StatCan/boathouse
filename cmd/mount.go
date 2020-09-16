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
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/StatCan/boathouse/internal/agent"
	"github.com/StatCan/boathouse/internal/client"
	"github.com/StatCan/boathouse/internal/flexvol"
	"github.com/StatCan/boathouse/internal/utils"
	"github.com/sevlyar/go-daemon"
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
		var err error
		var socketPath *net.UnixAddr

		// Setup a context that we will cancel when the process is requested to terminate
		ctx, cancel := context.WithCancel(context.Background())

		sigs := make(chan os.Signal, 1)
		done := make(chan bool, 1)

		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigs
			cancel()
			done <- true
		}()

		// 0. Decode options
		var options map[string]string
		err = json.Unmarshal([]byte(args[1]), &options)
		if err != nil {
			klog.Fatalf("failed to parse options: %v", err)
		}

		if flag := cmd.Flag("agent-socket-path"); flag != nil {
			socketPath, err = net.ResolveUnixAddr("unix", flag.Value.String())
			if err != nil {
				log.Fatalf("failed to resolve unix socket: %v", err)
			}
		}

		c, err := client.NewClient(socketPath)
		if err != nil {
			log.Fatalf("failed to create client: %v", err)
		}

		vaultPath := ""
		if val, ok := options["vault-path"]; ok {
			vaultPath = val
		}

		vaultTTL := time.Duration(0)
		if val, ok := options["vault-ttl"]; ok {
			dval, err := time.ParseDuration(val)
			if err != nil {
				klog.Warningf("failed to parse vault ttl duration: %v", err)
			} else {
				vaultTTL = dval
			}
		}

		// 1. Fork(ish)!
		// @TODO: Don't print success from parent until we know the child is good
		dctx := new(daemon.Context)
		child, err := dctx.Reborn()
		if err != nil {
			klog.Fatalf("failed to daemonize: %v", err)
		}

		// 4a. [Client] On success, signal parent that we have successfully started and write pid to state file
		// 4ai: [Client] Timeout until credentials expire. On expiry, request new credentials, terminate goofys and start goofys
		// 4b. [Client] On failure, signal parent that we have not started. Exit.
		// 5. [Parent] Exits and returns success/failure based on signal
		if child != nil {
			klog.Infof("child: %v", child)
			response := flexvol.DriverStatus{
				Status:  flexvol.StatusSuccess,
				Message: "Started disk mount",
			}

			err := utils.PrintJSON(os.Stdout, response)
			if err != nil {
				log.Fatal(err)
			}
			os.Exit(0)
		} else {
			defer dctx.Release()
		}

		// 2. Request credentials from the agent
	GoofysLoop:
		for {
			creds, err := c.IssueCredentials(agent.IssueCredentialRequest{
				Path: vaultPath,
				TTL:  vaultTTL,
			})
			if err != nil {
				klog.Errorf("failed to issue credentials: %v", err)
				time.Sleep(time.Second)
				continue
			}
			klog.Infof("obtained credentials from vault: %s, expires at %v", creds.AccessKey, creds.Lease.Expiry)

			// 3. [Client] Start goofys

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
			credscontext, _ := context.WithDeadline(ctx, creds.Lease.Expiry)

			goofys := exec.CommandContext(credscontext, "goofys", goofysArgs...)

			// Setup environment variable for access key and secret key
			goofys.Env = os.Environ()
			goofys.Env = append(goofys.Env, fmt.Sprintf("AWS_ACCESS_KEY_ID=%s", creds.AccessKey))
			goofys.Env = append(goofys.Env, fmt.Sprintf("AWS_SECRET_ACCESS_KEY=%s", creds.SecretKey))

			klog.Infof("starting goofys")
			err = goofys.Run()

			if err != nil {
				switch credscontext.Err() {
				case context.DeadlineExceeded:
					klog.Warningf("goofys terminated due to credential expiry")
				case context.Canceled:
					klog.Warningf("goofys terminated due to context cancellation")
					break GoofysLoop
				default:
					klog.Errorf("goofys terminated due to error: %v", err)
				}
			}

			klog.Infof("goofys terminated... restarting")
		}

		<-done
		klog.Infof("terminating")
	},
}

func init() {
	rootCmd.AddCommand(mountCmd)

	mountCmd.Flags().StringP("agent-socket-path", "a", "/tmp/boathouse.sock", "Address to connect to the agent.")
}
