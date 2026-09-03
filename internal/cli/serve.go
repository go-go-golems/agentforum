package cli

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-go-golems/agentforum/internal/config"
	"github.com/go-go-golems/agentforum/internal/server"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/types"
)

// --- serve: W7 single-binary server ------------------------------------

// ServeCommand runs the HTTP server (API + embedded UI when built with the
// `embed` tag) until interrupted. The listen address is a plain flag (env:
// AGENTFORUM_SERVE_LISTEN via the section prefix) and the database comes
// from the shared connection section like every other command.
type ServeCommand struct{ *cmds.CommandDescription }

// serveSettings is the serve command's own flag section.
type serveSettings struct {
	Listen string `glazed:"listen"`
}

const serveSectionSlug = "serve"

func newServeSection() (*schema.SectionImpl, error) {
	return schema.NewSection(
		serveSectionSlug, "Serve",
		schema.WithFields(
			fields.New("listen", fields.TypeString,
				fields.WithDefault("127.0.0.1:8080"),
				fields.WithHelp("Listen address for the HTTP server (env AGENTFORUM_SERVE_LISTEN)")),
		),
	)
}

// NewServeCommand builds the `agentforum serve` command.
func NewServeCommand() (*ServeCommand, error) {
	sec, err := newServeSection()
	if err != nil {
		return nil, err
	}
	conn, err := config.ConnectionSection()
	if err != nil {
		return nil, err
	}
	return &ServeCommand{CommandDescription: cmds.NewCommandDescription(
		"serve",
		cmds.WithShort("Run the agentforum HTTP server (API + web UI)"),
		cmds.WithLong(`Run the agentforum HTTP server: the /v1 API, a health check at /healthz, and (when the binary was built with the `+"`embed`"+` tag) the web UI at /.

The UI is embedded by:
  make build-web && go build -tags embed ./cmd/agentforum

Plain builds serve the API only. The listen address defaults to
127.0.0.1:8080 (AGENTFORUM_SERVE_LISTEN); the database comes from the
shared connection settings (AGENTFORUM_DB / --db).

Examples:
  agentforum serve --db /tmp/forum.db --listen 127.0.0.1:8080
  AGENTFORUM_DB=/tmp/forum.db AGENTFORUM_SERVE_LISTEN=:8080 agentforum serve`),
		cmds.WithSections(sec, conn),
	)}, nil
}

// RunIntoGlazeProcessor satisfies cmds.GlazeCommand.
func (c *ServeCommand) RunIntoGlazeProcessor(
	ctx context.Context,
	vals *values.Values,
	gp middlewares.Processor,
) error {
	conn, err := decodeConnection(vals)
	if err != nil {
		return err
	}
	var settings serveSettings
	if err := vals.DecodeSectionInto(serveSectionSlug, &settings); err != nil {
		return err
	}
	if settings.Listen == "" {
		settings.Listen = "127.0.0.1:8080"
	}

	svc, cleanup, err := openService(ctx, conn)
	if err != nil {
		return err
	}
	defer cleanup()

	srv := &http.Server{
		Addr:              settings.Listen,
		Handler:           server.New(svc),
		ReadHeaderTimeout: 10 * time.Second,
	}
	// Announce before blocking: glazed rows are only flushed when this
	// function returns, so the address goes straight to the terminal.
	fmt.Printf("agentforum serving http://%s (db: %s)\n",
		settings.Listen, config.ResolveDBPath(conn.DBPath))

	errCh := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	// Block until SIGINT/SIGTERM or the cobra context is done, then shut
	// down gracefully so long-polls drain (they observe the context).
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return fmt.Errorf("agentforum: serve: %w", err)
	case sig := <-sigCh:
		fmt.Printf("\nagentforum: received %s, shutting down…\n", sig)
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("agentforum: shutdown: %w", err)
	}
	return nil
}

// row helper silence (serve emits no rows; kept for symmetry with the
// package's five-part contract)
var _ = types.NewRow
