package service

import (
	"github.com/pentops/log.go/grpc_log"
	"github.com/pentops/log.go/log"
	"github.com/pentops/realms/j5auth"
	"google.golang.org/grpc"
)

func GRPCMiddleware() []grpc.UnaryServerInterceptor {
	return []grpc.UnaryServerInterceptor{
		grpc_log.UnaryServerInterceptor(log.DefaultContext, log.DefaultTrace, log.DefaultLogger),
		j5auth.GRPCMiddleware,
	}
}
