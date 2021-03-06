// Copyright © 2018 NAME HERE <EMAIL ADDRESS>
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

package cmd

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/CrowdSurge/banner"
	"github.com/blang/semver"
	"github.com/spf13/cobra"
)

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an EKS kubernetes cluster on AWS",
	Long: `
	Create an EKS cluster on AWS, the following binary needs to be in your PATH:

	    aws-iam-authenticator`,
	Run: func(cmd *cobra.Command, args []string) {

		if len(args) > 0 {
			fmt.Println()
			fmt.Println("Invalid, there is no need to use arguments with this command")
			fmt.Println()
			fmt.Println("Simple use : ekscluster create")
			fmt.Println()
			os.Exit(0)
		}

		banner.Print("kubernauts eks cli")

		fmt.Println()

		fmt.Println()

		kube, err := exec.LookPath("kubectl")
		if err != nil {
			log.Fatal("kubectl not found, kindly check")
		}
		fmt.Printf("Found kubectl at %s\n", kube)
		rr, err := exec.Command("kubectl", "version", "--client", "--short").Output()
		if err != nil {
			log.Fatal(err)
		}

		log.Println(string(rr))

		//Check if kubectl version is greater or equal to 1.10

		parts := strings.Split(string(rr), " ")

		KubeCtlVer := strings.Replace((parts[2]), "v", "", -1)

		v1, err := semver.Make("1.10.0")
		v2, err := semver.Make(strings.TrimSpace(KubeCtlVer))

		if v2.LT(v1) {
			log.Fatalln("kubectl client version on this system is less than the required version 1.10.0")
		}

		// Check if AWS authenticator binary is present in the working directory
		if _, err := exec.LookPath("aws-iam-authenticator"); err != nil {
			log.Fatalln("AWS Authenticator binary not found")
		}

		// Check if terraform binary is present in the working directory
		if _, err := os.Stat("./terraform"); err != nil {
			log.Fatalln("Terraform binary not found in the installation folder")
		}

		log.Println("Terraform binary exists in the installation folder, terraform version:")

		terr, err := exec.Command("./terraform", "version").Output()
		if err != nil {
			log.Fatal(err)
		}
		log.Println(string(terr))

		// Check if a terraform state file aclready exists
		if _, err := os.Stat("./terraform.tfstate"); err == nil {
			log.Fatalln("There is an existing cluster, please remove terraform.tfstate file or delete the installation before proceeding")
		}

		log.Println("Checking AWS Credentials")

		if os.Getenv("AWS_ACCESS_KEY_ID") == "" {
			log.Fatalln("AWS_ACCESS_KEY_ID not exported as environment variable, kindly check")
		}

		if os.Getenv("AWS_SECRET_ACCESS_KEY") == "" {
			log.Fatalln("AWS_SECRET_ACCESS_KEY not exported as environment variable, kindly check")
		}

		// Terraform Initialization and create the infrastructure

		log.Println("starting terraform init")

		terrInit := exec.Command("terraform", "init")
		terrInit.Dir = "./"
		out, _ := terrInit.StdoutPipe()
		terrInit.Start()
		scanInit := bufio.NewScanner(out)
		for scanInit.Scan() {
			m := scanInit.Text()
			fmt.Println(m)
		}

		terrInit.Wait()

		log.Println("starting terraform apply")
		terrSet := exec.Command("terraform", "apply", "-auto-approve")
		terrSet.Dir = "./"
		stdout, err := terrSet.StdoutPipe()
		terrSet.Stderr = terrSet.Stdout
		terrSet.Start()

		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			m := scanner.Text()
			fmt.Println(m)
		}

		terrSet.Wait()

		// Export KUBECONFIG file to the installation folder
		log.Println("Exporting kubeconfig file to the installation folder")

		kubeconf := exec.Command("terraform", "output", "kubeconfig")

		// open the out file for writing
		outfile, err := os.Create("./kubeconfig")
		if err != nil {
			panic(err)
		}
		defer outfile.Close()
		kubeconf.Stdout = outfile

		err = kubeconf.Start()
		if err != nil {
			panic(err)
		}
		kubeconf.Wait()

		log.Println("To use the kubeconfig file, do the following:")

		log.Println("export KUBECONFIG=~/.kubeconfig")

		// Output configmap to create Worker nodes

		log.Println("Exporting Worker nodes config-map to the installation folder")

		confmap := exec.Command("terraform", "output", "config-map")

		// open the out file for writing
		outconf, err := os.Create("./config-map-aws-auth.yaml")
		if err != nil {
			panic(err)
		}
		defer outconf.Close()
		confmap.Stdout = outconf

		err = confmap.Start()
		if err != nil {
			panic(err)
		}
		confmap.Wait()

		// Create Worker nodes usign the Configmap created above

		log.Println("Creating Worker Nodes")
		WorkerNodeSet := exec.Command("kubectl", "--kubeconfig", "./kubeconfig", "apply", "-f", "./config-map-aws-auth.yaml")
		WorkerNodeSet.Dir = "./"

		workerNodeOut, err := WorkerNodeSet.Output()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf(string(workerNodeOut))

		log.Println("Worker nodes are coming up one by one, it will take some time depending on the number of worker nodes you specified")

		os.Exit(0)

	},
}

func init() {
	rootCmd.AddCommand(createCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// createCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// createCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
