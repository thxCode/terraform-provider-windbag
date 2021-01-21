package dial

import (
	"context"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/sync/errgroup"

	"github.com/thxcode/terraform-provider-windbag/windbag/dial/powershell"
	"github.com/thxcode/terraform-provider-windbag/windbag/pki"
	"github.com/thxcode/terraform-provider-windbag/windbag/utils"
)

const sshKeepaliveSliding = 5 * time.Second

// SSHOptions specifies the options to dial SSH server.
type SSHOptions struct {
	Address           string
	Username          string
	Password          string
	KeyPEMBlockBytes  []byte
	CertPEMBlockBytes []byte
	WithAgent         bool
}

// DialSSH creates a dialer over SSH,
// which is inspired by rancher/rke tunnel.
func SSH(opts SSHOptions) (Dialer, error) {
	if opts.Address == "" {
		return nil, errors.New("cannot dial to SSH server as the address is blank")
	}

	if opts.Password == "" && len(opts.KeyPEMBlockBytes) == 0 {
		return nil, errors.New("cannot dial to SSH server as the authentication is incomplete")
	}

	var config, err = getSSHClientConfig(opts.Username, opts.Password, opts.KeyPEMBlockBytes, opts.CertPEMBlockBytes, opts.WithAgent)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create SSH client config")
	}

	cli, err := ssh.Dial("tcp", opts.Address, config)
	if err != nil {
		var errMsg = err.Error()
		if strings.Contains(errMsg, "no key found") {
			return nil, errors.Wrapf(err, "unable to dial SSH server with %s, please check if the configured key or specified key file is a valid SSH Private Key.", opts.Address)
		} else if strings.Contains(errMsg, "no supported methods remain") {
			return nil, errors.Wrapf(err, "unable to dial SSH server with %s, please check if you are able to SSH to the node using the specified SSH Private Key and if you have configured the correct SSH username.", opts.Address)
		} else if strings.Contains(errMsg, "cannot decode encrypted private keys") {
			return nil, errors.Wrapf(err, "unable to dial SSH server with %s, using encrypted private keys is only supported using ssh-agent, please configure to use the `SSH_AUTH_SOCK` environment variable.", opts.Address)
		} else if strings.Contains(errMsg, "operation timed out") {
			return nil, errors.Wrapf(err, "unable to dial SSH server with %s, please check if the node is up and is accepting SSH connections or check network policies and firewall rules.", opts.Address)
		}
		return nil, errors.Wrapf(err, "failed to dial SSH server with %s", opts.Address)
	}
	return &sshDialer{addr: opts.Address, cli: cli}, nil
}

type sshDialer struct {
	addr string
	cli  *ssh.Client
}

func (d sshDialer) DialContext(ctx context.Context, n, addr string) (conn net.Conn, err error) {
	var done = make(chan struct{}, 1)
	go func() {
		// NB(thxCode) this goroutine can leak for as long as the underlying Dialer implementation takes to timeout.
		conn, err = d.Dial(n, addr)
		close(done)
		if conn != nil && ctx.Err() != nil {
			_ = conn.Close()
		}
	}()
	select {
	case <-ctx.Done():
		err = ctx.Err()
	case <-done:
	}
	return conn, err
}

func (d sshDialer) Dial(n, addr string) (net.Conn, error) {
	return d.cli.Dial(n, addr)
}

func (d sshDialer) Close() error {
	return d.cli.Close()
}

func (d sshDialer) PowerShell(ctx context.Context, options *powershell.CreateOptions, interaction func(c context.Context, ps *powershell.PowerShell) error) error {
	if interaction == nil {
		log.Printf("[DEBUG] Skipped to interact with %s as the interaction is nil\n", d.addr)
		return nil
	}

	var s, err = d.cli.NewSession()
	if err != nil {
		log.Printf("[ERROR] Failed to create SSH session of %s: %v\n", d.addr, err)
		return err
	}
	defer func() {
		if err = s.Close(); err != nil && err != io.EOF {
			log.Printf("[ERROR] Failed to close SSH session of %s: %v\n", d.addr, err)
		}
	}()

	if ctx == nil {
		ctx = context.Background()
	}
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		// keepalive
		defer utils.HandleCrashSilent()
		var t *time.Ticker
		defer func() {
			if t != nil {
				t.Stop()
			}
		}()
		for {
			select {
			case <-ctx.Done():
				return nil
			default:
			}

			if _, err := s.SendRequest("keepalive", true, nil); err != nil {
				return err
			}
			log.Printf("[TRACE] Ping SSH session of %s\n", d.addr)

			if t == nil {
				t = time.NewTicker(sshKeepaliveSliding)
			} else {
				t.Reset(sshKeepaliveSliding)
			}

			select {
			case <-ctx.Done():
				return nil
			case <-t.C:
			}
		}
	})
	eg.Go(func() error {
		defer utils.HandleCrash()
		if err := interaction(ctx, powershell.Create(s, options)); err != nil {
			return err
		}
		// return EOF to close the keepalive goroutine
		return io.EOF
	})
	err = eg.Wait()
	if err == io.EOF {
		return nil
	}
	return err
}

func (d sshDialer) Copy(ctx context.Context, src io.Reader, dst string) (int64, error) {
	var cli, err = sftp.NewClient(d.cli)
	if err != nil {
		return 0, errors.Wrap(err, "failed to create SFTP client")
	}
	defer cli.Close()

	dstFile, err := cli.Create(dst)
	if err != nil {
		return 0, errors.Wrap(err, "failed to create destination file via SFTP client")
	}
	defer dstFile.Close()

	copied, err := io.Copy(dstFile, src)
	if err != nil {
		return copied, errors.Wrap(err, "failed to ship source file to destination via SFTP client")
	}
	return copied, nil
}

// getSSHClientConfig returns the SSH client config.
func getSSHClientConfig(username, password string, keyPem, certPem []byte, withAgent bool) (*ssh.ClientConfig, error) {
	var config = &ssh.ClientConfig{
		User:            username,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	// ssh-agent at first
	if withAgent {
		if sock := os.Getenv("SSH_AUTH_SOCK"); sock != "" {
			sshAgent, err := net.Dial("unix", sock)
			if err != nil {
				return nil, errors.Wrapf(err, "cannot connect to SSH agent socket %q", sock)
			}

			config.Auth = append(config.Auth, ssh.PublicKeysCallback(agent.NewClient(sshAgent).Signers))
			log.Println("[DEBUG] Using SSH_AUTH_SOCK: ", sock)
			return config, nil
		}
	}

	if len(keyPem) != 0 {
		var signer, err = pki.ParseSSHPrivateKey(keyPem)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse SSH private key")
		}

		if len(certPem) > 0 {
			ak, err := pki.ParseSSHAuthorizedKey(certPem)
			if err != nil {
				return nil, errors.Wrap(err, "failed to parse SSH certificate")
			}
			signer, err = ssh.NewCertSigner(ak, signer)
			if err != nil {
				return config, err
			}
		}

		config.Auth = append(config.Auth, ssh.PublicKeys(signer))
	}

	if password != "" {
		config.Auth = append(config.Auth, ssh.Password(password))
	}

	return config, nil
}
