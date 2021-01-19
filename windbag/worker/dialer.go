package worker

import (
	"context"
	"io"
	"net"

	"github.com/thxcode/terraform-provider-windbag/windbag/worker/powershell"
)

// Dialer specifies a closable dialer.
type Dialer interface {
	DialContext(ctx context.Context, n, addr string) (net.Conn, error)
	Dial(n, addr string) (net.Conn, error)
	Close() error
	PowerShell(ctx context.Context, opts *powershell.CreateOptions, interaction func(ctx context.Context, ps *powershell.PowerShell) error) error
	Copy(ctx context.Context, src io.Reader, dst string) (int64, error)
}
