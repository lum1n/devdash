package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/lum1n/devdash/internal/api"
	"github.com/lum1n/devdash/internal/core"
	"github.com/lum1n/devdash/internal/plugin"
	"github.com/lum1n/devdash/internal/plugin/acc"
	"github.com/lum1n/devdash/internal/plugin/gh"
	"github.com/lum1n/devdash/internal/plugin/ports"
	"github.com/lum1n/devdash/internal/plugin/tmux"
	"github.com/lum1n/devdash/internal/tui"
	"github.com/spf13/cobra"
)

var (
	cfgPath string
	jsonOut bool
)

// Execute runs the CLI. Bare `devdash` opens the TUI when stdout is a tty.
func Execute() error {
	root := &cobra.Command{
		Use:   "devdash",
		Short: "Local project dashboard — web + TUI",
		Long: `devdash scans configured roots for git repos and surfaces status,
shortcuts, and plugin widgets.

The Go process is the source of truth. TanStack Start in ./web talks to
the API from devdash serve.`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := newApp()
			if err != nil {
				return err
			}
			return tui.Run(app)
		},
	}
	root.PersistentFlags().StringVar(&cfgPath, "config", "", "path to config.yaml (default ~/.config/devdash/config.yaml)")
	root.PersistentFlags().BoolVar(&jsonOut, "json", false, "JSON output")
	root.AddCommand(serveCmd(), tuiCmd(), scanCmd(), versionCmd())
	return root.Execute()
}

func newApp() (*core.App, error) {
	reg := &plugin.Registry{}
	acc.Register(reg)
	tmux.Register(reg)
	ports.Register(reg)
	gh.Register(reg)
	return core.New(cfgPath, reg)
}

func serveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Start the local JSON API for the TanStack Start web UI",
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := newApp()
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := app.Scan(ctx); err != nil {
				fmt.Fprintf(os.Stderr, "scan: %v\n", err)
			}
			cfg := app.Config()
			srv := &http.Server{
				Addr:              cfg.Listen,
				Handler:           (&api.Server{App: app}).Handler(),
				ReadHeaderTimeout: 5 * time.Second,
			}
			ln, err := net.Listen("tcp", cfg.Listen)
			if err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "devdash api on http://%s\n", ln.Addr())
			fmt.Fprintf(os.Stderr, "web: cd web && pnpm dev  (proxies /api here)\n")
			return srv.Serve(ln)
		},
	}
}

func tuiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tui",
		Short: "Open the Bubble Tea dashboard",
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := newApp()
			if err != nil {
				return err
			}
			return tui.Run(app)
		},
	}
}

func scanCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "scan",
		Short: "Scan configured roots and print an overview",
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := newApp()
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := app.Scan(ctx); err != nil {
				return err
			}
			ov, err := app.Overview(ctx)
			if err != nil {
				return err
			}
			if jsonOut {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(ov)
			}
			fmt.Printf("roots %d  repos %d  dirty %d  behind %d  ahead %d\n",
				len(ov.Roots), ov.Counts.Repos, ov.Counts.Dirty, ov.Counts.Behind, ov.Counts.Ahead)
			for _, r := range ov.Repos {
				mark := " "
				if r.Attention {
					mark = "*"
				}
				fmt.Printf("%s %-22s %-12s dirty=%-5v %s\n", mark, r.Name, r.Branch, r.Dirty, r.Stack)
			}
			return nil
		},
	}
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("devdash 0.1.0")
		},
	}
}
