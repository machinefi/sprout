package main

import (
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/pkg/errors"

	"github.com/machinefi/sprout/clients"
	"github.com/machinefi/sprout/cmd/enode/api"
	"github.com/machinefi/sprout/cmd/enode/config"
	"github.com/machinefi/sprout/datasource"
	"github.com/machinefi/sprout/p2p"
	"github.com/machinefi/sprout/persistence"
	"github.com/machinefi/sprout/project"
	"github.com/machinefi/sprout/task"
)

var conf *config.Config

func main() {
	var err error
	conf, err = config.Get()
	if err != nil {
		panic(errors.Wrap(err, "failed to init enode config"))
	}
	conf.Print()
	slog.Info("enode config loaded")

	persistence, err := persistence.NewPostgres(conf.DatabaseDSN)
	if err != nil {
		log.Fatal(err)
	}

	nextTaskID, err := persistence.FetchNextTaskID()
	if err != nil {
		log.Fatal(err)
	}

	_ = clients.NewManager()

	projectManager, err := project.NewManager(conf.ChainEndpoint, conf.ProjectContractAddress, conf.ProjectFileDirectory, conf.ProjectCacheDirectory, conf.IPFSEndpoint)
	if err != nil {
		log.Fatal(err)
	}

	datasource, err := datasource.NewPostgres(conf.DatasourceDSN)
	if err != nil {
		log.Fatal(err)
	}

	dispatcher, err := task.NewDispatcher(persistence, projectManager, conf.OperatorPrivateKey, conf.OperatorPrivateKeyED25519)
	if err != nil {
		log.Fatal(err)
	}

	networking, err := p2p.NewPubSubs(projectManager, dispatcher, conf.BootNodeMultiAddr, conf.IoTeXChainID)
	if err != nil {
		log.Fatal(err)
	}

	go task.Dispatching(nextTaskID, datasource, networking)

	go func() {
		if err := api.NewHttpServer(persistence, conf).Run(conf.ServiceEndpoint); err != nil {
			log.Fatal(err)
		}
	}()

	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)
	<-done
}
