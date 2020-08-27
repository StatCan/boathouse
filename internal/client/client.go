package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"

	"github.com/StatCan/boathouse/internal/agent"
	"k8s.io/klog"
)

// Client is a boathouse client.
type Client struct {
	sock *net.UnixAddr
}

// NewClient generates a new Boathouse client.
func NewClient(sock *net.UnixAddr) (*Client, error) {
	return &Client{
		sock: sock,
	}, nil
}

func (c Client) IssueCredentials(req agent.IssueCredentialRequest) (*agent.IssueCredentialResponse, error) {
	// Make an HTTP request to the unix socket
	transport := http.Transport{
		Dial: func(proto, addr string) (conn net.Conn, err error) {
			return net.Dial("unix", c.sock.Name)
		},
	}
	httpClient := http.Client{
		Transport: &transport,
	}

	b, err := json.Marshal(req)
	if err != nil {
		klog.Errorf("failed to marshal json: %v", err)
		return nil, err
	}

	resp, err := httpClient.Post("http://boathouse/issue", "application/json", bytes.NewBuffer(b))
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		klog.Errorf("error reading response body: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	var creds agent.IssueCredentialResponse
	err = json.Unmarshal(body, &creds)
	if err != nil {
		klog.Errorf("error unmarshalling json: %v", err)
		return nil, err
	}

	return &creds, nil
}
