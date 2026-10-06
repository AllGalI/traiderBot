package httpdebug

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"
)

type Server struct {
	server *http.Server
}

func NewServer(
	addr string,
	handler http.Handler,
) *Server {
	return &Server{
		server: &http.Server{
			Addr:              addr,
			Handler:           handler,
			ReadHeaderTimeout: 5 * time.Second,
		},
	}
}

func (s *Server) Run(ctx context.Context) error {
	shutdownDone := make(chan struct{})

	go func() {
		defer close(shutdownDone)

		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(
			context.Background(),
			5*time.Second,
		)
		defer cancel()

		if err := s.server.Shutdown(shutdownCtx); err != nil {
			log.Printf(
				"debug HTTP shutdown error: %v",
				err,
			)
		}
	}()

	err := s.server.ListenAndServe()

	<-shutdownDone

	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}

	return err
}