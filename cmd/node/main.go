package main

import (
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"

	"github.com/machinefi/w3bstream-mainnet/msg"
	"github.com/machinefi/w3bstream-mainnet/msg/handler"
	"github.com/machinefi/w3bstream-mainnet/vm"
)

func main() {
	var programLevel = slog.LevelDebug
	h := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: programLevel})
	slog.SetDefault(slog.New(h))

	dbMigrate()
	viper.MustBindEnv("ENDPOINT")
	viper.MustBindEnv("RISC0_SERVER_ENDPOINT")
	viper.MustBindEnv("HALO2_SERVER_ENDPOINT")
	viper.MustBindEnv("PROJECT_CONFIG_FILE")
	viper.MustBindEnv("CHAIN_ENDPOINT")
	viper.MustBindEnv("OPERATOR_PRIVATE_KEY")
	viper.MustBindEnv("DATABASE_URL")

	vmHandler := vm.NewHandler(
		map[vm.Type]string{
			vm.Risc0: viper.Get("RISC0_SERVER_ENDPOINT").(string),
			vm.Halo2: viper.Get("HALO2_SERVER_ENDPOINT").(string),
		},
	)
	msgHandler := handler.New(vmHandler, viper.Get("CHAIN_ENDPOINT").(string), viper.Get("OPERATOR_PRIVATE_KEY").(string), viper.Get("PROJECT_CONFIG_FILE").(string))

	router := gin.Default()
	router.POST("/message", func(c *gin.Context) {
		var req msgReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, newErrResp(err))
			return
		}
		msg := &msg.Msg{
			ProjectID:      req.ProjectID,
			ProjectVersion: req.ProjectVersion,
			Data:           req.Data,
		}
		slog.Debug("received your message, handling")
		if err := msgHandler.Handle(msg); err != nil {
			c.JSON(http.StatusInternalServerError, newErrResp(err))
			return
		}

		c.Status(http.StatusOK)
	})

	go func() {
		if err := router.Run(viper.Get("ENDPOINT").(string)); err != nil {
			log.Fatal(err)
		}
	}()

	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)
	<-done
}
