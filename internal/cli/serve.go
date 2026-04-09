package cli

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/DotNaos/moodle-cli/internal/api"
	"github.com/spf13/cobra"
)

var serveAddr string
var serveShutdownTimeout time.Duration
var serveSchool string
var serveUsername string
var servePassword string

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the REST API server",
	Long:  "Start a long-running HTTP server that exposes Moodle data as JSON over a REST API.",
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return nil, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		runtimeLoginOverrides = loginInputOverrides{
			School:   serveSchool,
			Username: serveUsername,
			Password: servePassword,
		}

		if err := ensureServeSession(); err != nil {
			return err
		}

		router, err := api.NewRouter(api.ServerOptions{
			ClientProvider: func() (api.Client, error) {
				return ensureAuthenticatedClient()
			},
		})
		if err != nil {
			return err
		}

		server := &http.Server{
			Addr:              serveAddr,
			Handler:           router,
			ReadHeaderTimeout: 10 * time.Second,
		}

		fmt.Printf("Starting Moodle API server on %s\n", serveAddr)

		errCh := make(chan error, 1)
		go func() {
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				errCh <- err
			}
		}()

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

		select {
		case err := <-errCh:
			return err
		case sig := <-sigCh:
			fmt.Printf("Received %s, shutting down...\n", sig)
		}

		ctx, cancel := context.WithTimeout(context.Background(), serveShutdownTimeout)
		defer cancel()
		return server.Shutdown(ctx)
	},
}

func init() {
	defaultAddr := ":8080"
	if port := strings.TrimSpace(os.Getenv("PORT")); port != "" {
		if strings.HasPrefix(port, ":") {
			defaultAddr = port
		} else {
			defaultAddr = ":" + port
		}
	}

	serveCmd.Flags().StringVar(&serveAddr, "addr", defaultAddr, "Address to bind the API server to (e.g. :8080 or 127.0.0.1:8080)")
	serveCmd.Flags().DurationVar(&serveShutdownTimeout, "shutdown-timeout", 10*time.Second, "Grace period for graceful shutdown")
	serveCmd.Flags().StringVar(&serveSchool, "school", "", "School id override used for a fresh login. Only fhgr is currently active; multi-school support is not active")
	serveCmd.Flags().StringVar(&serveUsername, "username", "", "Username/email used for a fresh login before starting the server")
	serveCmd.Flags().StringVar(&servePassword, "password", "", "Password used for a fresh login before starting the server")

	serveCmd.RegisterFlagCompletionFunc("school", completeSchoolIDs)
}
