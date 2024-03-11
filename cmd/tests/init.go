package tests

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/pkg/errors"

	"github.com/machinefi/sprout/clients"
	"github.com/machinefi/sprout/cmd/enode/api"
	enodeconfig "github.com/machinefi/sprout/cmd/enode/config"
	znodeconfig "github.com/machinefi/sprout/cmd/znode/config"
	"github.com/machinefi/sprout/persistence"
	"github.com/machinefi/sprout/project"
	"github.com/machinefi/sprout/task"
	"github.com/machinefi/sprout/types"
	"github.com/machinefi/sprout/vm"
)

var (
	env = "INTEGRATION_TEST"
)

func init() {
	_ = os.Setenv("ZNODE_ENV", env)
	_ = os.Setenv("ENODE_ENV", env)

	znodeconf, err := znodeconfig.Get()
	if err != nil {
		os.Exit(-1)
	}
	enodeconf, err := enodeconfig.Get()
	if err != nil {
		os.Exit(-1)
	}

	go runZnode(znodeconf)
	go runEnode(enodeconf)

	// repeat 3 and duration 3s
	if err := checkLiveness(3, 5, func() error {
		if _, e := http.Get(fmt.Sprintf("http://localhost%s/live", enodeconf.ServiceEndpoint)); err != nil {
			return e
		}
		return nil
	}); err != nil {
		slog.Error("http server failed to start", "error", err)
	}
}

func checkLiveness(repeat int, duration int64, ping func() error) error {
	for i := 0; i < repeat; i++ {
		if err := ping; err != nil {
			slog.Warn("retry again", "duration", duration, "error", err)
			time.Sleep(time.Duration(duration) * time.Second)
			continue
		}
		break
	}
	return nil
}

func migrateDatabase(dsn string) error {
	var schema = `
	CREATE TABLE IF NOT EXISTS vms (
		id SERIAL PRIMARY KEY,
		project_name VARCHAR NOT NULL,
		elf TEXT NOT NULL,
		image_id VARCHAR NOT NULL
	  );
	  
	  CREATE TABLE IF NOT EXISTS proofs (
		id SERIAL PRIMARY KEY,
		image_id VARCHAR NOT NULL,
		private_input VARCHAR NOT NULL,
		public_input VARCHAR NOT NULL,
		receipt_type VARCHAR NOT NULL,
		receipt TEXT,
		status VARCHAR NOT NULL,
		create_at TIMESTAMP NOT NULL DEFAULT now()
	  );`

	slog.Debug("connecting database", "dsn", dsn)
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return errors.Wrap(err, "connect db failed")
	}
	if _, err = db.Exec(schema); err != nil {
		return errors.Wrap(err, "migrate db failed")
	}
	return nil
}

func runZnode(conf *znodeconfig.Config) {
	if err := migrateDatabase(conf.DatabaseDSN); err != nil {
		log.Fatal(err)
	}

	vmHandler := vm.NewHandler(
		map[types.VM]string{
			types.VMRisc0:  conf.Risc0ServerEndpoint,
			types.VMHalo2:  conf.Halo2ServerEndpoint,
			types.VMZkwasm: conf.ZKWasmServerEndpoint,
		},
	)

	projectManager, err := project.NewManager(conf.ChainEndpoint, conf.ProjectContractAddress, conf.ProjectFileDirectory, conf.IPFSEndpoint)
	if err != nil {
		log.Fatal(err)
	}

	taskProcessor, err := task.NewProcessor(vmHandler, projectManager, conf.BootNodeMultiAddr, conf.IoTeXChainID)
	if err != nil {
		log.Fatal(err)
	}

	taskProcessor.Run()

	slog.Info("znode started")
}

func runEnode(conf *enodeconfig.Config) {
	pg, err := persistence.NewPostgres(conf.DatabaseDSN)
	if err != nil {
		log.Fatal(err)
	}

	_ = clients.NewManager()

	projectManager, err := project.NewManager(conf.ChainEndpoint, conf.ProjectContractAddress, conf.ProjectFileDirectory, conf.IPFSEndpoint)
	if err != nil {
		log.Fatal(err)
	}

	dispatcher, err := task.NewDispatcher(pg, projectManager, conf.BootNodeMultiAddr, conf.OperatorPrivateKey, conf.OperatorPrivateKeyED25519, conf.IoTeXChainID)
	if err != nil {
		log.Fatal(err)
	}
	go dispatcher.Dispatch()

	go func() {
		if err := api.NewHttpServer(pg, projectManager, conf).Run(conf.ServiceEndpoint); err != nil {
			log.Fatal(err)
		}
	}()

	slog.Info("znode started")
}
