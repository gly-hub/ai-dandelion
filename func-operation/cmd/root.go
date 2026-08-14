package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/team-dandelion/ai-dandelion/func-operation/boot"
	"github.com/team-dandelion/ai-dandelion/func-operation/global"
	"github.com/team-dandelion/ai-dandelion/func-operation/index"
	"github.com/team-dandelion/quickgo/logger"
)

var rootCmd = &cobra.Command{
	Use:              "server",
	TraverseChildren: true,
	Short:            "start func-operation grpc server",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		logger.Info(context.Background(), "func-operation pre run called")
		err := boot.Boot(global.ConfigPath)
		if err != nil {
			logger.Error(context.Background(), "boot err: %v", err)
			global.ErrChan <- fmt.Errorf("boot err: %v", err)
			return err
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		logger.Info(context.Background(), "func-operation server called")
		if global.GetApp().GrpcServer() == nil {
			global.ErrChan <- fmt.Errorf("grpc server init failed")
			return
		}
		if global.GetApp().GrpcClientManager() == nil {
			global.ErrChan <- fmt.Errorf("grpc client init failed")
			return
		}
		for _, serviceName := range []string{"ai-agent", "system"} {
			if err := global.GetApp().GrpcClientManager().RegisterService(serviceName); err != nil {
				global.ErrChan <- fmt.Errorf("register %s service err: %v", serviceName, err)
				return
			}
		}
		if err := global.GetApp().GrpcServer().RegisterService(index.RegisterHandler); err != nil {
			global.ErrChan <- fmt.Errorf("register service err: %v", err)
			return
		}
		if err := global.GetApp().Start(); err != nil {
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
				index.StopArtifactRuntime()
				index.StopOutboxRuntime()
				global.GetApp().Stop()
				return
			case s := <-c:
				log.Printf("get a signal: %s\n", s.String())
				index.StopArtifactRuntime()
				index.StopOutboxRuntime()
				global.GetApp().Stop()
				return
			}
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&global.ConfigPath, "conf", "c", "", "Config file path")
}
