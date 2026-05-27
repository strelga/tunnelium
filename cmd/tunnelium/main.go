package main

import (
	"fmt"
	"os"
	"strings"

	"tunnelium/src/gost/socks"
	"tunnelium/src/paths"
	"tunnelium/src/service"
	"tunnelium/src/update"

	"github.com/spf13/cobra"
)

var version = "dev"

func main() {
	rootCmd := &cobra.Command{
		Use:   "tunnelium",
		Short: "VPN service manager",
		Long:  "VPN service manager for managing gost VPN and proxy services.",
	}

	rootCmd.SetVersionTemplate("tunnelium {{.Version}}\n")
	rootCmd.Version = version

	// --- completion ---
	completionCmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion scripts",
		Long: `Generate shell completion scripts for tunnelium.

To load completions:

Bash:

	 # Load completions in your current shell session
	 source <(tunnelium completion bash)

	 # Run this once to install completions for all new sessions (macOS with Homebrew):
	 tunnelium completion bash > $(brew --prefix)/etc/bash_completion.d/tunnelium

	 # Run this once to install completions for all new sessions (Linux, bash-completion v2):
	 tunnelium completion bash | sudo tee /usr/share/bash-completion/completions/tunnelium > /dev/null

Zsh:

	 # Load completions in your current shell session
	 source <(tunnelium completion zsh)

	 # If shell completion is not already enabled in your environment,
	 # you will need to enable it. You can execute the following once:
	 echo "autoload -U compinit; compinit" >> ~/.zshrc

	 # Then, run this once to install the completion file:
	 tunnelium completion zsh > "${fpath[1]}/_tunnelium"

	 # You will need to start a new shell for this setup to take effect.

Fish:

	 # Load completions in your current shell session
	 tunnelium completion fish | source

	 # Run this once to install completions for all new sessions:
	 tunnelium completion fish > ~/.config/fish/completions/tunnelium.fish

PowerShell:

	 # Load completions in your current shell session
	 tunnelium completion powershell | Out-String | Invoke-Expression

	 # Run this once to install completions for all new sessions:
	 tunnelium completion powershell > tunnelium.ps1
	 # and source this file from your PowerShell profile.
`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return cmd.Root().GenBashCompletion(os.Stdout)
			case "zsh":
				return cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				return cmd.Root().GenFishCompletion(os.Stdout, true)
			case "powershell":
				return cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
			}
			return nil
		},
	}
	rootCmd.AddCommand(completionCmd)

	// --- self-update ---
	selfUpdateCmd := &cobra.Command{
		Use:   "self-update",
		Short: "Update tunnelium to the latest version from GitHub",
		Long:  "Download and replace the current binary with the latest release from GitHub.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return update.Run(version)
		},
	}
	rootCmd.AddCommand(selfUpdateCmd)

	// --- service ---
	serviceCmd := &cobra.Command{
		Use:   "service",
		Short: "Service management commands",
	}
	rootCmd.AddCommand(serviceCmd)

	// --- service add ---
	var (
		serviceType  string
		instanceName string
		port         int
		// Gost-specific flags
		gostRole        string
		nextHopHost     string
		nextHopPort     int
		socksPort       int
		httpPort        int
		gostTLSCertPath string
	)

	serviceAddCmd := &cobra.Command{
		Use:   "add",
		Short: "Add a new service",
		Long: `Add a new service. If no flags provided and TTY available, runs in interactive mode.

For gost client: --type, --name, --role=client, --next-hop-host are required.
  At least one of --socks-port or --http-port is required.
For gost server: --type, --name, --role=server, --port are required.

Examples:
	 tunnelium service add --type gost --name incoming --role client --socks-port 1081 --next-hop-host 192.0.2.10
	 tunnelium service add --type gost --name incoming --role client --socks-port 1081 --http-port 8080 --next-hop-host 192.0.2.10
	 tunnelium service add --type gost --name relay-eu --role server --port 443`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// No flags → interactive mode (if TTY)
			if !cmd.Flags().Changed("type") && !cmd.Flags().Changed("name") {
				params, err := service.RunInteractive()
				if err != nil {
					return err
				}
				if err := service.Add(*params); err != nil {
					return err
				}
				printServiceAdded(params)
				return nil
			}

			if serviceType == "" || instanceName == "" {
				return fmt.Errorf("--type and --name are required")
			}

			params := service.ServiceParams{
				ServiceType:     service.ServiceType(serviceType),
				InstanceName:    instanceName,
				HostSystemPort:  port,
				GostRole:        service.GostRole(gostRole),
				GostNextHopHost: nextHopHost,
				GostNextHopPort: nextHopPort,
				GostSocksPort:   socksPort,
				GostHTTPPort:    httpPort,
				GostTLSCertPath: gostTLSCertPath,
			}

			if err := service.Add(params); err != nil {
				return err
			}

			printServiceAdded(&params)
			return nil
		},
	}

	serviceAddCmd.Flags().StringVar(&serviceType, "type", "", "Service type (gost)")
	serviceAddCmd.Flags().StringVar(&instanceName, "name", "", "Instance name (e.g. incoming, cross-dc)")
	serviceAddCmd.Flags().IntVar(&port, "port", 0, "Host system port (required for gost server)")
	serviceAddCmd.Flags().StringVar(&gostRole, "role", "", "Gost role: client or server")
	serviceAddCmd.Flags().StringVar(&nextHopHost, "next-hop-host", "", "Next hop host (gost client)")
	serviceAddCmd.Flags().IntVar(&nextHopPort, "next-hop-port", 443, "Next hop port (gost client, default 443)")
	serviceAddCmd.Flags().IntVar(&socksPort, "socks-port", 0, "SOCKS5+auth port on host (gost client)")
	serviceAddCmd.Flags().IntVar(&httpPort, "http-port", 0, "HTTP proxy port on host (gost client)")
	serviceAddCmd.Flags().StringVar(&gostTLSCertPath, "tls-cert", "", "Path to combined PEM file (gost server, optional)")

	serviceCmd.AddCommand(serviceAddCmd)

	// --- service start ---
	serviceStartCmd := &cobra.Command{
		Use:   "start <name>",
		Short: "Start a service",
		Long:  "Start a service by name (e.g. gost-incoming). Runs docker compose up -d.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := service.Start(args[0]); err != nil {
				return err
			}
			fmt.Printf("Service %q started\n", args[0])
			return nil
		},
	}
	serviceCmd.AddCommand(serviceStartCmd)

	// --- service stop ---
	serviceStopCmd := &cobra.Command{
		Use:   "stop <name>",
		Short: "Stop a service",
		Long:  "Stop a service by name (e.g. gost-incoming). Runs docker compose stop.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := service.Stop(args[0]); err != nil {
				return err
			}
			fmt.Printf("Service %q stopped\n", args[0])
			return nil
		},
	}
	serviceCmd.AddCommand(serviceStopCmd)

	// --- service restart ---
	serviceRestartCmd := &cobra.Command{
		Use:   "restart <name>",
		Short: "Restart a service",
		Long:  "Restart a service by name (e.g. gost-incoming). Runs docker compose restart.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := service.Restart(args[0]); err != nil {
				return err
			}
			fmt.Printf("Service %q restarted\n", args[0])
			return nil
		},
	}
	serviceCmd.AddCommand(serviceRestartCmd)

	// --- gost ---
	gostCmd := &cobra.Command{
		Use:   "gost",
		Short: "Gost service management",
	}
	rootCmd.AddCommand(gostCmd)

	// --- gost socks-user ---
	gostSocksUserCmd := &cobra.Command{
		Use:   "socks-user",
		Short: "SOCKS5 user management for gost instances",
	}
	gostCmd.AddCommand(gostSocksUserCmd)

	// --- gost socks-user create ---
	gostSocksUserCreateCmd := &cobra.Command{
		Use:   "create <instance> [username] [password]",
		Short: "Create a new SOCKS5 user",
		Long:  "Create a new user for a gost SOCKS5 instance. Random password is generated if omitted.",
		Args:  cobra.RangeArgs(1, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			instance := args[0]
			c := socks.NewInstance(instance)

			username := ""
			password := ""
			if len(args) > 1 {
				username = args[1]
			}
			if len(args) > 2 {
				password = args[2]
			}
			result, err := c.CreateUser(username, password)
			if err != nil {
				return err
			}
			fmt.Printf("Created user %s with password %s\n", result.Username, result.Password)
			return nil
		},
	}
	gostSocksUserCmd.AddCommand(gostSocksUserCreateCmd)

	// --- gost socks-user remove ---
	gostSocksUserRemoveCmd := &cobra.Command{
		Use:   "remove <instance> <username>",
		Short: "Remove a SOCKS5 user",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := socks.NewInstance(args[0])
			return c.RemoveUser(args[1])
		},
	}
	gostSocksUserCmd.AddCommand(gostSocksUserRemoveCmd)

	// --- gost socks-user list ---
	gostSocksUserListCmd := &cobra.Command{
		Use:   "list <instance>",
		Short: "List all SOCKS5 users for a gost instance",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := socks.NewInstance(args[0])
			users, err := c.ListUsers()
			if err != nil {
				return err
			}

			if len(users) == 0 {
				fmt.Println("No users found")
				return nil
			}

			for _, u := range users {
				fmt.Printf("%s=%s\n", u.Username, u.Password)
			}
			return nil
		},
	}
	gostSocksUserCmd.AddCommand(gostSocksUserListCmd)

	// --- gost reload ---
	gostReloadCmd := &cobra.Command{
		Use:   "reload <instance>",
		Short: "Reload gost configuration (SIGHUP)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := socks.NewInstance(args[0])
			return c.ReloadConfig()
		},
	}
	gostCmd.AddCommand(gostReloadCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func printServiceAdded(params *service.ServiceParams) {
	serviceName := fmt.Sprintf("%s-%s", params.ServiceType, params.InstanceName)
	fmt.Printf("Service %q added successfully!\n", serviceName)
	fmt.Printf("  Container: tunnelium-%s\n", serviceName)
	fmt.Printf("  Config:    %s/\n", paths.ServiceDir(serviceName))

	command := service.GenerateGostCommand(*params)
	fmt.Printf("  Command:   gost %s\n", strings.Join(command, " "))

	if params.GostRole == service.GostRoleServer {
		fmt.Printf("  TLS cert:  %s\n", paths.ServiceDir(serviceName)+"/tls.pem")
		fmt.Printf("  Port:      %d\n", params.HostSystemPort)
	} else {
		if params.GostSocksPort > 0 {
			fmt.Printf("  SOCKS:     :%d (with auth)\n", params.GostSocksPort)
		}
		if params.GostHTTPPort > 0 {
			fmt.Printf("  HTTP:      :%d\n", params.GostHTTPPort)
		}
	}

	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  1. Start:       tunnelium service start %s\n", serviceName)
	if params.GostRole == service.GostRoleClient && params.GostSocksPort > 0 {
		fmt.Printf("  2. Add users:   tunnelium gost socks-user create %s <username>\n", params.InstanceName)
	}
}
