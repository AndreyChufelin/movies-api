package auth

import (
	"context"
	"fmt"
	"net"

	"github.com/AndreyChufelin/movies-api/internal/logger"
	"github.com/AndreyChufelin/movies-api/internal/storage"
	pbuser "github.com/AndreyChufelin/movies-auth/pkg/pb/user"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type Auth struct {
	logger *logger.Logger
	client pbuser.UserServiceClient
	addr   string
	conn   *grpc.ClientConn
}

func NewAuth(log *logger.Logger, host, port string) *Auth {
	return &Auth{
		logger: log,
		addr:   net.JoinHostPort(host, port),
	}
}

func (a *Auth) Start() error {
	conn, err := grpc.NewClient(a.addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		a.logger.Error("failed to connect to auth grpc")
		return err
	}

	a.client = pbuser.NewUserServiceClient(conn)
	a.conn = conn

	return nil
}

func (a *Auth) Close() error {
	err := a.conn.Close()
	if err != nil {
		return fmt.Errorf("failed to connect to grpc client: %w", err)
	}

	return nil
}

func (a *Auth) Verify(ctx context.Context, token string) (*storage.User, error) {
	u, err := a.client.VerifyToken(ctx, &pbuser.VerifyTokenRequest{
		Token: token,
	})
	if err != nil {
		if grpcErr, ok := status.FromError(err); ok {
			if grpcErr.Code() == codes.Unauthenticated || grpcErr.Code() == codes.InvalidArgument {
				a.logger.Warn("invalid token")
				return nil, storage.ErrInvalidToken
			}
		}
		a.logger.Error("failed to verify token", "error", err)
		return nil, fmt.Errorf("failed to verify token: %w", err)
	}

	user := &storage.User{
		ID:          u.Id,
		Activated:   u.Activated,
		Permissions: u.Permissions,
	}
	return user, nil
}
