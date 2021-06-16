package dmsgpty

import (
	"context"
	"fmt"
	"net"
	"os"
	"runtime"

	"github.com/sirupsen/logrus"
	"github.com/skycoin/skycoin/src/util/logging"
	terminal "golang.org/x/term"

	"github.com/skycoin/dmsg/cipher"
)

// CLI connects with and has ownership over a dmsgpty.Host.
type CLI struct {
	Log  logrus.FieldLogger `json:"-"`
	Net  string             `json:"cli_network"`
	Addr string             `json:"cli_address"`
}

// DefaultCLI returns a CLI with default values.
func DefaultCLI() CLI {
	return CLI{
		Log:  logging.MustGetLogger("dmsgpty-cli"),
		Net:  DefaultCLINet,
		Addr: DefaultCLIAddr,
	}
}

// WhitelistClient returns a client that interacts with the Host's whitelist.
func (cli *CLI) WhitelistClient() (*WhitelistClient, error) {
	conn, err := cli.prepareConn()
	if err != nil {
		return nil, err
	}
	return NewWhitelistClient(conn)
}

// StartLocalPty starts a pty on the host.
func (cli *CLI) StartLocalPty(ctx context.Context, cmd string, args ...string) error {
	conn, err := cli.prepareConn()
	if err != nil {
		return err
	}

	ptyC, err := NewPtyClient(conn)
	if err != nil {
		return err
	}

	if runtime.GOOS != "Windows" {
		restore, err := cli.prepareStdin()
		if err != nil {
			return err
		}
		defer restore()
	}

	return cli.servePty(ctx, ptyC, cmd, args)
}

// StartRemotePty starts a pty on a remote host, proxied via the local pty.
func (cli *CLI) StartRemotePty(ctx context.Context, rPK cipher.PubKey, rPort uint16, cmd string, args ...string) error {
	conn, err := cli.prepareConn()
	if err != nil {
		return err
	}

	ptyC, err := NewProxyClient(conn, rPK, rPort)
	if err != nil {
		return err
	}

	if runtime.GOOS != "Windows" {
		restore, err := cli.prepareStdin()
		if err != nil {
			return err
		}
		defer restore()
	}

	return cli.servePty(ctx, ptyC, cmd, args)
}

// prepareConn prepares a connection with the dmsgpty-host.
func (cli *CLI) prepareConn() (net.Conn, error) {

	// Set defaults.
	if cli.Log == nil {
		cli.Log = logging.MustGetLogger("dmsgpty-cli")
	}
	if cli.Net == "" {
		cli.Net = DefaultCLINet
	}
	if cli.Addr == "" {
		cli.Addr = DefaultCLIAddr
	}

	cli.Log.
		WithField("address", fmt.Sprintf("%s://%s", cli.Net, cli.Addr)).
		Infof("Requesting...")

	conn, err := net.Dial(cli.Net, cli.Addr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to dmsgpty-host: %v", err)
	}
	return conn, nil
}

// prepareStdin sets stdin to raw mode and provides a function to restore the original state.
func (cli *CLI) prepareStdin() (restore func(), err error) {
	var oldState *terminal.State
	if oldState, err = terminal.MakeRaw(int(os.Stdin.Fd())); err != nil {
		cli.Log.
			WithError(err).
			Warn("Failed to set stdin to raw mode.")
		return
	}
	restore = func() {
		// Attempt to restore state.
		if err := terminal.Restore(int(os.Stdin.Fd()), oldState); err != nil {
			cli.Log.
				WithError(err).
				Error("Failed to restore original stdin state.")
		}
	}
	return
}

