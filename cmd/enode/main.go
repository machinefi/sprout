package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/viper"

	"github.com/machinefi/sprout/clients"
	"github.com/machinefi/sprout/cmd/enode/api"
	"github.com/machinefi/sprout/persistence"
	"github.com/machinefi/sprout/project"
	"github.com/machinefi/sprout/task"
)

func main() {
	initLogger()
	initConfig()

	pg, err := persistence.NewPostgres(viper.GetString(DatabaseDSN))
	if err != nil {
		log.Fatal(err)
	}

	_ = clients.NewManager()

	projectManager, err := project.NewManager(viper.GetString(ChainEndpoint), viper.GetString(ProjectContractAddress), viper.GetString(ProjectFileDirectory), viper.GetString(IPFSEndpoint))
	if err != nil {
		log.Fatal(err)
	}

	dispatcher, err := task.NewDispatcher(pg, projectManager, viper.GetString(BootNodeMultiaddr), viper.GetString(OperatorPrivateKey), viper.GetString(OperatorPrivateKeyED25519), viper.GetInt(IotexChainID))
	if err != nil {
		log.Fatal(err)
	}
	go dispatcher.Dispatch()

	go func() {
		if err := api.NewHttpServer(pg, viper.GetString(DIDAuthServerEndpoint), projectManager, getENodeConfig).Run(viper.GetString(HttpServiceEndpoint)); err != nil {
			log.Fatal(err)
		}
	}()

	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)
	<-done
}
