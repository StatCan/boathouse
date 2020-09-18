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
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/StatCan/boathouse/internal/agent"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/spf13/cobra"

	vault "github.com/hashicorp/vault/api"
)

// agentCmd represents the agent command
var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Runs the boathouse agent",
	Long:  `The boathouse agent provides a proxy to obtain resources from HashiCorp Vault.`,
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		var socketPath *net.UnixAddr

		// Process the socket
		if flag := cmd.Flag("socket-path"); flag != nil {
			socketPath, err = net.ResolveUnixAddr("unix", flag.Value.String())
			if err != nil {
				log.Fatalf("failed to resolve unix socket: %v", err)
			}
		}

		// If the socket exists, lets remove it
		if err = os.Remove(socketPath.Name); !os.IsNotExist(err) && err != nil {
			log.Fatalf("failed to remove existing socket: %v", err)
		}

		// Start a new listener
		listener, err := net.ListenUnix(socketPath.Net, socketPath)
		if err != nil {
			log.Fatalf("failed to setup listener: %v", err)
		}
		defer listener.Close()

		// Setup a webserver
		router := mux.NewRouter()

		// Agent
		vc, err := vault.NewClient(&vault.Config{
			Address:      os.Getenv("VAULT_ADDR"),
			AgentAddress: os.Getenv("VAULT_AGENT_ADDR"),
		})
		agent, err := agent.NewAgent(vc)
		if err != nil {
			log.Fatalf("failed to create vault client: %v", err)
		}

		if token := os.Getenv("VAULT_TOKEN"); token != "" {
			vc.SetToken(token)
		}

		router.Path("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("hello world"))
		})

		router.Path("/issue").HandlerFunc(agent.HandleIssueCredentials)

		server := http.Server{
			Handler:      handlers.CombinedLoggingHandler(os.Stdout, router),
			WriteTimeout: 15 * time.Second,
			ReadTimeout:  15 * time.Second,
		}

		log.Printf("listening on %v", socketPath)
		log.Fatal(server.Serve(listener))
	},
}

func init() {
	rootCmd.AddCommand(agentCmd)

	agentCmd.Flags().StringP("socket-path", "s", path.Join(os.TempDir(), "boathouse.sock"), "Listen address for agent communication.")
}
