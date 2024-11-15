package mux

import (
	"context"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
	"net/http"
	"net/http/pprof"
)

type Server struct {
	httpServer *http.Server
}

func New() *Server {
	return &Server{}
}

func (s *Server) Run(addr string) error {
	mux := http.NewServeMux()
	// metrics
	mux.Handle("/metrics", promhttp.Handler())
	// pprof routes
	mux.Handle("/debug/pprof/", http.HandlerFunc(pprof.Index))
	mux.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
	mux.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
	mux.Handle("/handle/pprof/symbol", http.HandlerFunc(pprof.Symbol))
	mux.Handle("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))
	mux.Handle("/debug/pprof/block", pprof.Handler("block"))
	mux.Handle("/debug/pprof/heap", pprof.Handler("heap"))
	mux.Handle("/debug/pprof/allocs", pprof.Handler("allocs"))
	mux.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
	mux.Handle("/debug/pprof/threadcreate", pprof.Handler("goroutine"))

	s.httpServer = &http.Server{
		Addr:           addr,
		Handler:        mux,
		MaxHeaderBytes: 1 << 20,
	}

	log.Info().Msgf("HTTP Run server on %s", addr)

	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
