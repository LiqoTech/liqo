package discovery

import (
	"context"
	"errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
	"os"
	"strings"
)

type TxtData struct {
	ID               string
	Namespace        string
	AllowUntrustedCA bool
	ApiUrl           string
}

func (txtData TxtData) Encode() ([]string, error) {
	res := []string{
		"id=" + txtData.ID,
		"namespace=" + txtData.Namespace,
		"untrusted-ca=" + txtData.GetAllowUntrustedCA(),
		"url=" + txtData.ApiUrl,
	}
	return res, nil
}

func (txtData *TxtData) GetAllowUntrustedCA() string {
	if txtData.AllowUntrustedCA {
		return "true"
	} else {
		return "false"
	}
}

func Decode(address string, port string, data []string) (*TxtData, error) {
	var res = TxtData{}
	for _, d := range data {
		if strings.HasPrefix(d, "id=") {
			res.ID = d[len("id="):]
		} else if strings.HasPrefix(d, "namespace=") {
			res.Namespace = d[len("namespace="):]
		} else if strings.HasPrefix(d, "untrusted-ca=") {
			res.AllowUntrustedCA = d[len("untrusted-ca="):] == "true"
		} else if strings.HasPrefix(d, "url=") {
			// used in LAN discovery
			res.ApiUrl = d[len("url="):]
		}
	}
	// used in WAN discovery
	if address != "" && port != "" {
		res.ApiUrl = "https://" + address + ":" + port
	}
	if res.ID == "" || res.Namespace == "" || res.ApiUrl == "" {
		return nil, errors.New("TxtData missing required field")
	}
	return &res, nil
}

func (discovery *DiscoveryCtrl) GetTxtData() (*TxtData, error) {
	apiUrl, err := discovery.GetAPIUrl()
	if err != nil {
		klog.Error(err)
		return nil, err
	}
	return &TxtData{
		ID:               discovery.ClusterId.GetClusterID(),
		Namespace:        discovery.Namespace,
		AllowUntrustedCA: discovery.Config.AllowUntrustedCA,
		ApiUrl:           apiUrl,
	}, nil
}

// get API Server Url for this cluster
// if APISERVER env variable is set we read it's ip form this variable
//     (this can be useful on managed k8s services where we have no master node)
// else get the IP of first master
// if APISERVER_PORT env variable is set we use it has port
// else we fallback to default port
func (discovery *DiscoveryCtrl) GetAPIUrl() (string, error) {
	address, ok := os.LookupEnv("APISERVER")
	if !ok || address == "" {
		nodes, err := discovery.crdClient.Client().CoreV1().Nodes().List(context.TODO(), v1.ListOptions{
			LabelSelector: "node-role.kubernetes.io/master",
		})
		if err != nil {
			return "", err
		}
		if len(nodes.Items) == 0 || len(nodes.Items[0].Status.Addresses) == 0 {
			err = errors.New("no APISERVER env variable found and no master node found, one of the two values must be present")
			klog.Error(err)
			return "", err
		}
		address = nodes.Items[0].Status.Addresses[0].Address
	}

	port, ok := os.LookupEnv("APISERVER_PORT")
	if !ok {
		port = "6443"
	}

	return "https://" + address + ":" + port, nil
}
