package cmd

import (
	"context"
	"fmt"
	"log"
	"os/signal"
	"syscall"

	"github.com/gly-hub/ai-dandelion/inner-gateway/boot"
	"github.com/gly-hub/ai-dandelion/inner-gateway/global"
	"github.com/gly-hub/ai-dandelion/inner-gateway/index"
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
	PreRun: func(cmd *cobra.Command, args []string) {
		logger.Info(context.Background(), "server pre run called")
		err := boot.Boot(global.ConfigPath)
		if err != nil {
			logger.Error(context.Background(), "boot err: %v", err)
			global.ErrChan <- fmt.Errorf("boot err: %v", err)
			return
		}
	},

	Run: func(cmd *cobra.Command, args []string) {
		logger.Info(context.Background(), "server called")

		if global.GetApp().GrpcClientManager() == nil {
			global.ErrChan <- fmt.Errorf("grpc client init failed")
			return
		}

		// 注册需要调用的 gRPC 服务
		for _, serviceName := range []string{"ai-agent", "func-operation", "system"} {
			err := global.GetApp().GrpcClientManager().RegisterService(serviceName)
			if err != nil {
				global.ErrChan <- fmt.Errorf("register service %s err: %v", serviceName, err)
				return
			}
		}

		if global.GetApp().HTTPServer() == nil {
			global.ErrChan <- fmt.Errorf("http server init failed")
			return
		}

		err := global.GetApp().HTTPServer().RegisterApp(index.RouteHandler)
		if err != nil {
			global.ErrChan <- fmt.Errorf("register app err: %v", err)
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
				global.GetApp().Stop()
				return
			case s := <-c:
				log.Printf("get a signal: %s\n", s.String())
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
