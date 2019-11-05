// Copyright 2019 Antrea Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package antctl

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
)

var isAgent = strings.HasPrefix(os.Getenv("POD_NAME"), "antrea-agent")

// RequestOption contains options to do a request to CLI server.
type RequestOption struct {
	Kubeconfig string
	Name       string
	Args       map[string]string
}

// URI returns the request path of the request option.
func (opt *RequestOption) URI(kubeconfig *rest.Config) *url.URL {
	u, _ := url.Parse(kubeconfig.Host)
	u.Path = path.Join("/apis", kubeconfig.GroupVersion.Group, kubeconfig.GroupVersion.Version, opt.Name)

	q := u.Query()
	for k, v := range opt.Args {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()

	return u
}

// Client does request to a CLI server and get the response.
type Client struct {
	GroupVersion *schema.GroupVersion
	Codec        serializer.CodecFactory
}

func (c *Client) resolveKubeconfig(opt *RequestOption) (*rest.Config, error) {
	kubeconfig, cfgErr := clientcmd.BuildConfigFromFlags("", opt.Kubeconfig)
	if cfgErr != nil {
		klog.Infof("Can not use kubeconfig %s, trying to use in-cluster config: %v", opt.Kubeconfig, cfgErr)

		var err error
		kubeconfig, err = rest.InClusterConfig()
		if err != nil {
			klog.Infof("Can not use in-cluster config: %v", err)
			return nil, cfgErr
		}

		// In-pod mode, need to specify the port.
		var port int
		if isAgent {
			port = DefaultAgentLocalPort
		} else {
			port = DefaultControllerLocalPort
		}

		u := url.URL{
			Scheme: "http",
			Host:   net.JoinHostPort("127.0.0.1", fmt.Sprint(port)),
		}
		// Local mode just trust the sever and skip cert verification.
		kubeconfig.Host = u.String()
		kubeconfig.Insecure = true
		kubeconfig.CAFile = ""
		kubeconfig.CAData = nil
	}

	kubeconfig.GroupVersion = c.GroupVersion
	kubeconfig.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: c.Codec}

	return kubeconfig, nil
}

// Do makes the request by the command and the request option.
func (c *Client) Do(cmd *cobra.Command, opt *RequestOption) (io.Reader, error) {
	kubeconfig, err := c.resolveKubeconfig(opt)
	if err != nil {
		return nil, err
	}

	restClient, err := rest.RESTClientFor(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("Failed to create rest client: %w", err)
	}

	// If timeout is not set, no timeout.
	restClient.Client.Timeout, _ = cmd.Flags().GetDuration("timeout")

	uri := opt.URI(kubeconfig).String()
	klog.Infof("Requesting URI %s", uri)
	result := restClient.Get().RequestURI(uri).Do()
	if result.Error() != nil {
		return nil, fmt.Errorf("Error when requesting URI %s: %w", uri, result.Error())
	}
	raw, err := result.Raw()
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(raw), nil
}

// Init initializes the root command.
func Init(root *cobra.Command) {
	client := new(Client)
	Definition.ApplyToClient(client)
	Definition.ApplyToRootCommand(root, client, isAgent)
}
