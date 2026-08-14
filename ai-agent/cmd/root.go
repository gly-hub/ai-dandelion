package cmd

import (
	"context"
	"fmt"
	"log"
	"os/signal"
	"syscall"

	"github.com/gly-hub/ai-dandelion/ai-agent/boot"
	"github.com/gly-hub/ai-dandelion/ai-agent/global"
	"github.com/gly-hub/ai-dandelion/ai-agent/index"
	"github.com/gly-hub/quickgo/logger"
	"github.com/spf13/cobra"

	"os"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:              "server",
	TraverseChildren: true,
	Short:            "default cmd, start http、grpc server",
	Long:             ``,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		logger.Info(context.Background(), "server pre run called")
		err := boot.Boot(global.ConfigPath)
		if err != nil {
			logger.Error(context.Background(), "boot err: %v", err)
			global.ErrChan <- fmt.Errorf("boot err: %v", err)
			return err
		}
		return nil
	},

	Run: func(cmd *cobra.Command, args []string) {
		logger.Info(context.Background(), "server called")
		if global.GetApp().GrpcServer() == nil {
			global.ErrChan <- fmt.Errorf("grpc server init failed")
			return
		}

		err := global.GetApp().GrpcServer().RegisterService(index.RegisterHandler)
		if err != nil {
			global.ErrChan <- fmt.Errorf("register service err: %v", err)
			return
		}

		err = global.GetApp().Start()
		if err != nil {
			logger.Error(context.Background(), "start err: %v", err)
			global.ErrChan <- fmt.Errorf("start err: %v", err)
			return
		}
	},

	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
		for {
			select {
			case err := <-global.ErrChan:
				if err == nil {
					continue
				}
				log.Printf("get global error: %v\n", err)
				index.StopAgentBotRuntime()
				global.GetApp().Stop()
				return
			case s := <-c:
				log.Printf("get a signal: %s\n", s.String())
				index.StopAgentBotRuntime()
				global.GetApp().Stop()
				return
			}
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&global.ConfigPath, "conf", "c", "", "Config file path")
	return
}
