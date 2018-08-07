package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"text/template"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

const (
	PluginName    = "cni-bridge"
	PluginVersion = "0.0.1"
)

var (
	bridgeName  = "cbr0"
	networkName = "cni-bridge"

	bridgeConfPath   = "/etc/cni/net.d/bridge.conf"
	loopbackConfPath = "/etc/cni/net.d/loopback.conf"

	bridgeTpl = template.Must(template.New("bridge-conf").Parse(`{
  "cniVersion": "0.1.0",
  "name": "{{.netName}}",
  "type": "bridge",
  "bridge": "{{.bridgeName}}",
  "mtu": 1460,
  "addIf": "eth0",
  "isGateway": true,
  "ipMasq": false,
  "hairpinMode": true,
  "forceAddress": true,
  "ipam": {
    "type": "host-local",
    "subnet": "{{.subnet}}",
    "gateway": "{{.gateway}}",
    "routes": [
      { "dst": "0.0.0.0/0" }
    ]
  }
}`))

	loopbackConf = `{
  "cniVersion": "0.1.0",
  "name": "cni-loopback",
  "type": "loopback"
}`

	rootCmd = cobra.Command{
		Use: "cni-bridge",
		RunE: func(cmd *cobra.Command, args []string) error {
			nodeName := os.Getenv("CURRENT_HOST_NODENAME")
			if nodeName == "" {
				return errors.New("can not find node name")
			}

			kubeConfig, err := rest.InClusterConfig()
			if err != nil {
				return err
			}

			client, err := kubernetes.NewForConfig(kubeConfig)
			if err != nil {
				return err
			}

			lw := &cache.ListWatch{
				ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
					return client.CoreV1().Nodes().List(options)
				},
				WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
					return client.CoreV1().Nodes().Watch(options)
				},
			}

			setPodCidrAndGateway := func(cidr *net.IPNet) error {
				createTempFileWithData := func(data []byte) (string, error) {
					tmpFile, err := ioutil.TempFile("/tmp", fmt.Sprintf("%s-tmp", PluginName))
					if err != nil {
						return "", err
					}
					defer tmpFile.Close()
					_, err = tmpFile.Write(data)
					if err != nil {
						return "", err
					}

					return tmpFile.Name(), nil
				}

				path, err := createTempFileWithData([]byte(loopbackConf))
				if err != nil {
					return err
				}

				if output, err := exec.Command("/bin/mv", path, loopbackConfPath).CombinedOutput(); err != nil {
					logrus.Errorf("%s, output: %s", err, output)
					return err
				}

				buf := bytes.Buffer{}

				networkNo := make([]byte, len(cidr.IP))

				for idx := range networkNo {
					networkNo[idx] = cidr.IP[idx] & cidr.Mask[idx]
				}

				networkNo[len(networkNo)-1] = 1

				gwIp := net.IP(networkNo)

				err = bridgeTpl.Execute(
					&buf,
					map[string]string{
						"netName":    networkName,
						"bridgeName": bridgeName,
						"subnet":     cidr.String(),
						"gateway":    gwIp.String(),
					},
				)
				if err != nil {
					return err
				}

				path, err = createTempFileWithData(buf.Bytes())
				if err != nil {
					return err
				}

				if output, err := exec.Command("/bin/mv", path, bridgeConfPath).CombinedOutput(); err != nil {
					logrus.Errorf("%s, output: %s", err, output)
					return err
				}
				return nil
			}

			_, controller := cache.NewInformer(
				lw,
				&v1.Node{},
				0,
				cache.ResourceEventHandlerFuncs{
					AddFunc: func(obj interface{}) {
						node := obj.(*v1.Node)
						if node.Name == nodeName {
							if node.Spec.PodCIDR == "" {
								logrus.Warningf("node %s has no cidr assigned, skipped", node.Name)
								return
							}
							_, cidr, err := net.ParseCIDR(node.Spec.PodCIDR)
							if err != nil {
								logrus.Errorln(err)
								return
							}

							if err := setPodCidrAndGateway(cidr); err != nil {
								logrus.Errorln(err)
								return
							}
						}
					},
					UpdateFunc: func(oldObj, newObj interface{}) {
						node := newObj.(*v1.Node)
						if node.Name == nodeName {
							if node.Spec.PodCIDR == "" {
								logrus.Warningf("node %s has no cidr assigned, skipped", node.Name)
								return
							}
							_, cidr, err := net.ParseCIDR(node.Spec.PodCIDR)
							if err != nil {
								logrus.Errorln(err)
								return
							}

							if err := setPodCidrAndGateway(cidr); err != nil {
								logrus.Errorln(err)
								return
							}
						}
					},
				},
			)

			controller.Run(make(<-chan struct{}, 0))

			return nil
		},
	}
)

func init() {
	rootCmd.Flags().StringVar(&bridgeName, "bridge-name", bridgeName, "")
	rootCmd.Flags().StringVar(&networkName, "network-name", networkName, "")

	rootCmd.Flags().StringVar(&bridgeConfPath, "bridge-conf-path", bridgeConfPath, "")
	rootCmd.Flags().StringVar(&loopbackConfPath, "loopback-conf-path", loopbackConfPath, "")
}

func main() {
	rootCmd.Execute()
}
