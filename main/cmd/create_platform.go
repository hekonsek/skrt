package cmd

import (
	"encoding/base64"
	"fmt"
	"github.com/GeertJohan/go.rice"
	"github.com/spf13/cobra"
	"html/template"
	"io/ioutil"
	"os"
	"os/exec"
	"time"
)

var ingressIP string
var ingressNodeSelector string

func init() {
	createPlatformCmd.Flags().StringVar(&ingressIP, "ingressIP", "", "IP address of ingress node.")
	createPlatformCmd.Flags().StringVar(&ingressNodeSelector, "ingressNodeSelector", "machine0001", "Node selector of ingress pod.")
	rootCmd.AddCommand(createPlatformCmd)
}

var createPlatformCmd = &cobra.Command{
	Use:   "create platform",
	Short: "Create skrt platform.",
	Long:  `Create skrt platform.`,
	Run: func(cmd *cobra.Command, args []string) {
		templateBox, err := rice.FindBox("templates")
		ExitOnError(err)

		skrtPlatform := NewSkrtPlatform()
		err = os.MkdirAll(skrtPlatform.Sandbox, 0700)
		if err != nil {
			fmt.Println(err)
			return
		}
		err = os.MkdirAll(skrtPlatform.Sandbox+"/templates", 0700)
		if err != nil {
			fmt.Println(err)
			return
		}

		chart, err := templateBox.Bytes("Chart.yaml")
		if err != nil {
			fmt.Println(err)
			return
		}
		err = ioutil.WriteFile(skrtPlatform.Sandbox+"/Chart.yaml", chart, 0644)
		if err != nil {
			fmt.Println(err)
			return
		}

		requirements, err := templateBox.Bytes("requirements.yaml")
		if err != nil {
			fmt.Println(err)
			return
		}
		err = ioutil.WriteFile(skrtPlatform.Sandbox+"/requirements.yaml", requirements, 0644)
		if err != nil {
			fmt.Println(err)
			return
		}

		values, err := templateBox.String("values.yml")
		if err != nil {
			fmt.Println(err)
			return
		}
		valuesTemplate, err := template.New("valuesTemplate").Parse(values)
		if err != nil {
			fmt.Println(err)
			return
		}
		valuesFile, err := os.Create(skrtPlatform.Sandbox + "/values.yml")
		if err != nil {
			fmt.Println(err)
			return
		}
		if ingressIP == "" {
			fmt.Println("Ingress IP cannot be empty. Please use --ingressIP option.")
			os.Exit(-1)
		}

		valuesx := map[string]map[string]interface{}{
			"Ingress": {
				"ExternalIp":   ingressIP,
				"NodeSelector": ingressNodeSelector,
			},
			"Prometheus": {
				"Host": fmt.Sprintf("prometheus.%s.nip.io", ingressIP),
			},
		}
		err = valuesTemplate.Execute(valuesFile, valuesx)
		ExitOnError(err)

		xxx := exec.Command("htpasswd", "-c", "-b", skrtPlatform.Sandbox+"/auth", "admin", "admin")
		x, _ := xxx.CombinedOutput()
		println(string(x))

		valuesxx, err := templateBox.String("secret-ingress-auth-prometheus.yml")
		if err != nil {
			fmt.Println(err)
			return
		}
		prometheusIngressAuthTemplate, err := template.New("prometheusIngressAuthTemplate").Parse(valuesxx)
		if err != nil {
			fmt.Println(err)
			return
		}
		prometheusIngressAuthFile, err := os.Create(skrtPlatform.Sandbox + "/templates/secret-ingress-auth-prometheus.yml")
		if err != nil {
			fmt.Println(err)
			return
		}
		auth, err := ioutil.ReadFile(skrtPlatform.Sandbox + "/auth")
		ExitOnError(err)
		valuesx["Prometheus"]["Ingress"] = map[string]interface{}{
			"Auth": base64.StdEncoding.EncodeToString(auth),
		}
		err = prometheusIngressAuthTemplate.Execute(prometheusIngressAuthFile, valuesx)
		ExitOnError(err)

		xxx = exec.Command("helm", "dependency", "update", skrtPlatform.Sandbox)
		x, _ = xxx.CombinedOutput()
		println(string(x))

		xxx = exec.Command("helm", "install", "--namespace=kube-system", "--name=skrt", skrtPlatform.Sandbox, "--values="+skrtPlatform.Sandbox+"/values.yml")
		x, _ = xxx.CombinedOutput()
		println(string(x))

		println(skrtPlatform.Sandbox)
	},
}

type SkrtPlatform struct {
	Sandbox string
}

func NewSkrtPlatform() SkrtPlatform {
	return SkrtPlatform{
		Sandbox: fmt.Sprintf("/tmp/skrt_%d", time.Now().Unix()),
	}
}

// Helper

func ExitOnError(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}