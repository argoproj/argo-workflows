package cmd

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/user"
	"path"
	"strings"

	"applatix.io/api"
	"applatix.io/axerror"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh/terminal"
)

var (
	loginConfig   string // --config
	clusterConfig api.ClusterConfig
)

func init() {
	RootCmd.AddCommand(loginCmd)

	loginCmd.Flags().StringVar(&loginConfig, "config", "", "Configuration name")
	loginCmd.Flags().StringVar(&clusterConfig.URL, "url", "", "Cluster URL")
	loginCmd.Flags().StringVar(&clusterConfig.Username, "username", "", "Username")
	loginCmd.Flags().StringVar(&clusterConfig.Password, "password", "", "Password")
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Configure or update an Argo configuration",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) >= 1 {
			clusterConfig.URL = args[0]
		}
		reader := bufio.NewReader(os.Stdin)

		if loginConfig == "" {
			fmt.Printf("Enter a configuration name (%s): ", api.DefaultConfigName)
			loginConfig, _ = reader.ReadString('\n')
			loginConfig = strings.TrimSpace(loginConfig)
			if loginConfig == "" {
				loginConfig = api.DefaultConfigName
			}
		}

		if clusterConfig.URL == "" {
			fmt.Printf("Enter cluster URL: ")
			clusterConfig.URL, _ = reader.ReadString('\n')
			clusterConfig.URL = strings.TrimSpace(clusterConfig.URL)
			if clusterConfig.URL == "" {
				log.Fatalln("Cluster URL required")
			}
		}
		clusterConfig.URL = strings.TrimRight(clusterConfig.URL, "/")

		if clusterConfig.Username == "" {
			fmt.Printf("Enter cluster username: ")
			clusterConfig.Username, _ = reader.ReadString('\n')
			clusterConfig.Username = strings.TrimSpace(clusterConfig.Username)
			if clusterConfig.Username == "" {
				log.Fatalln("Cluster username required")
			}
		}

		if clusterConfig.Password == "" {
			fmt.Printf("Enter cluster password: ")
			loginPassword, _ := terminal.ReadPassword(0)
			fmt.Println()
			clusterConfig.Password = string(loginPassword)
			if clusterConfig.Password == "" {
				log.Fatalln("Cluster password required")
			}
		}
		client := api.NewArgoClient(clusterConfig)
		_, axErr := client.Login()
		if axErr != nil {
			// TODO: need to completely rework axerror to preserve original,
			// underlying error and not rely on this clunky string search
			if axErr.Code == axerror.ERR_AX_HTTP_CONNECTION.Code && (strings.Contains(axErr.Message, "certificate is not valid") || strings.Contains(axErr.Message, "certificate is valid for")) {
				fmt.Printf("Cluster is using an invalid or self-signed certificate. Proceed insecurely (y/n)? ")
				insecure, _ := reader.ReadString('\n')
				insecure = strings.TrimSpace(strings.ToLower(insecure))
				if insecure != "y" && insecure != "yes" {
					os.Exit(1)
				}
				// Try again insecurely
				newTrue := true
				clusterConfig.Insecure = &newTrue
				client = api.NewArgoClient(clusterConfig)
				_, axErr = client.Login()
				if axErr != nil {
					log.Fatalln(axErr)
				}
			} else {
				log.Fatalln(axErr)
			}
		}
		client.Logout()
		usr, err := user.Current()
		if err != nil {
			log.Fatalln(err)
		}
		configPath := path.Join(usr.HomeDir, api.ArgoDir, loginConfig)
		err = clusterConfig.WriteConfigFile(configPath)
		if err != nil {
			log.Fatalf("Failed to write config file: %v\n", err)
		}
		log.Printf("Config written to: %s\n", configPath)
	},
}
