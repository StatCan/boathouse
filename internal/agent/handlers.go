package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"k8s.io/klog"

	vault "github.com/hashicorp/vault/api"
)

// HandleIssueCredentials issues credentials from an HTTP request
func (a *Agent) HandleIssueCredentials(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		klog.Errorf("error reading body: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	var req IssueCredentialRequest
	err = json.Unmarshal(body, &req)
	if err != nil {
		klog.Errorf("error decoding body: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	creds, err := a.IssueCredentials(r.Context(), req)
	if err != nil { // TODO: set status code based on error. Ex. 404 for not found
		klog.Errorf("error issuing credentials: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	b, err := json.Marshal(creds)
	if err != nil {
		klog.Errorf("error writing json: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(b)
}

// IssueCredentials issues the requested credentials
func (a *Agent) IssueCredentials(ctx context.Context, req IssueCredentialRequest) (*IssueCredentialResponse, error) {
	var creds *vault.Secret
	var err error

	klog.Infof("issuing credentials: %s with TTL %v", req.Path, req.TTL)

	if req.TTL > 0 {
		creds, err = a.vault.Logical().Read(req.Path)
	} else {
		creds, err = a.vault.Logical().ReadWithData(req.Path, map[string][]string{
			"ttl": []string{strconv.FormatInt(req.TTL.Microseconds(), 10)},
		})
	}
	if err != nil {
		klog.Warningf("unable to obtain MinIO token at %s: %v", req.Path, err)
		return nil, err
	}

	if creds == nil {
		return nil, fmt.Errorf("failure: no response returned from vault")
	}

	response := IssueCredentialResponse{
		Lease: Lease{
			ID:       creds.LeaseID,
			Duration: time.Duration(creds.LeaseDuration) * time.Second,
		},
	}

	if val, ok := creds.Data["accessKeyId"]; ok {
		response.AccessKey = val.(string)
	}

	if val, ok := creds.Data["secretAccessKey"]; ok {
		response.SecretKey = val.(string)
	}

	klog.Infof("issued credentials: %s, expiring in %v", response.AccessKey, response.Lease.Duration)

	return &response, nil
}
