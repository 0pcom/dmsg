// package main cmd/dmsgsocks/dmsgsocks.go
package main

import (
	"context"
	"log"
	"os"
	"net/http"
	cc "github.com/ivanpirog/coloredcobra"
	"github.com/skycoin/skywire-utilities/pkg/buildinfo"
	"github.com/skycoin/skywire-utilities/pkg/cipher"
	"github.com/skycoin/skywire-utilities/pkg/cmdutil"
	"github.com/skycoin/skywire-utilities/pkg/logging"
	"github.com/skycoin/skywire-utilities/pkg/skyenv"
	"github.com/spf13/cobra"
	"github.com/skycoin/dmsg/pkg/disc"


	dmsg "github.com/skycoin/dmsg/pkg/dmsg"
	"golang.org/x/net/proxy"
)

var (
	sk       cipher.SecKey
	dmsgDisc string
	dmsgPort uint
)

func init() {
	rootCmd.Flags().UintVarP(&dmsgPort, "port", "p", 1080, "dmsg port to serve SOCKS5 proxy")
	rootCmd.Flags().StringVarP(&dmsgDisc, "dmsg-disc", "D", "", "dmsg discovery url default:\n"+skyenv.DmsgDiscAddr)
	if os.Getenv("DMSGSOCKS_SK") != "" {
		sk.Set(os.Getenv("DMSGSOCKS_SK"))
	}
	rootCmd.Flags().VarP(&sk, "sk", "s", "a random key is generated if unspecified\n\r")
	var helpflag bool
	rootCmd.SetUsageTemplate(help)
	rootCmd.PersistentFlags().BoolVarP(&helpflag, "help", "h", false, "help for "+rootCmd.Use)
	rootCmd.SetHelpCommand(&cobra.Command{Hidden: true})
	rootCmd.PersistentFlags().MarkHidden("help")
}

var rootCmd = &cobra.Command{
	Use:   "dmsgsocks",
	Short: "dmsgsocks SOCKS5 proxy server",
	Long: `
	┌┬┐┌┬┐┌─┐┌─┐┌─┐┌─┐┌─┐┬┌─┌─┐
	 │││││└─┐│ ┬└─┐│ ││  ├┴┐└─┐
	─┴┘┴ ┴└─┘└─┘└─┘└─┘└─┘┴ ┴└─┘`,
	SilenceErrors:         true,
	SilenceUsage:          true,
	DisableSuggestions:    true,
	DisableFlagsInUseLine: true,
	Version:               buildinfo.Version(),
	PreRun: func(cmd *cobra.Command, args []string) {
		if dmsgDisc == "" {
			dmsgDisc = skyenv.DmsgDiscAddr
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		log := logging.MustGetLogger("dmsgsocks")

		ctx, cancel := cmdutil.SignalContext(context.Background(), log)
		defer cancel()
		pk, err := sk.PubKey()
		if err != nil {
			pk, sk = cipher.GenerateKeyPair()
		}

		// Create a new dmsg client
		c := dmsg.NewClient(pk, sk, disc.NewHTTP(dmsgDisc, &http.Client{}, log), dmsg.DefaultConfig())
		defer func() {
			if err := c.Close(); err != nil {
				log.WithError(err).Error()
			}
		}()

		go c.Serve(context.Background())

		select {
		case <-ctx.Done():
			log.WithError(ctx.Err()).Warn()
			return

		case <-c.Ready():
		}

		// Start serving the SOCKS5 proxy server
		go func() {
			<-ctx.Done()
			if err := c.Close(); err != nil {
				log.WithError(err).Error()
			}
		}()

		log.WithField("dmsg_addr", pk.String()).
			Info("SOCKS5 Proxy Server started.")

		// Create a SOCKS5 proxy server using the dmsg client
		proxyServer, err := proxy.SOCKS5("tcp", c.LocalAddr().String(), nil, proxy.Direct)
		if err != nil {
			log.WithError(err).Fatal("Failed to create SOCKS5 proxy server.")
		}

		// Start the SOCKS5 proxy server
		go func() {
			err := proxyServer.ListenAndServe()
			if err != nil {
				log.WithError(err).Fatal("Failed to start SOCKS5 Proxy Server.")
			}
		}()

		// Wait for the context to be canceled
		<-ctx.Done()

		log.Info("Stopping SOCKS5 Proxy Server.")
		proxyServer.Close()
	},
}

// Execute executes root CLI command.
func Execute() {
	cc.Init(&cc.Config{
		RootCmd:       rootCmd,
		Headings:      cc.HiBlue + cc.Bold, //+ cc.Underline,
		Commands:      cc.HiBlue + cc.Bold,
		CmdShortDescr: cc.HiBlue,
		Example:       cc.HiBlue + cc.Italic,
		ExecName:      cc.HiBlue + cc.Bold,
		Flags:         cc.HiBlue + cc.Bold,
		//FlagsDataType: cc.HiBlue,
		FlagsDescr:      cc.HiBlue,
		NoExtraNewlines: true,
		NoBottomNewline: true,
	})
	if err := rootCmd.Execute(); err != nil {
		log.Fatal("Failed to execute command: ", err)
	}
}

const help = "Usage:\r\n" +
	"  {{.UseLine}}{{if .HasAvailableSubCommands}}{{end}} {{if gt (len .Aliases) 0}}\r\n\r\n" +
	"{{.NameAndAliases}}{{end}}{{if .HasAvailableSubCommands}}\r\n\r\n" +
	"Available Commands:{{range .Commands}}{{if (or .IsAvailableCommand)}}\r\n  " +
	"{{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}\r\n\r\n" +
	"Flags:\r\n" +
	"{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}\r\n\r\n" +
	"Global Flags:\r\n" +
	"{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}\r\n\r\n"

func main() {
	Execute()
}
