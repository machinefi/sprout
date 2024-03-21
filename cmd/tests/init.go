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
	"github.com/machinefi/sprout/datasource"
	"github.com/machinefi/sprout/p2p"
	"github.com/machinefi/sprout/persistence"
	"github.com/machinefi/sprout/project"
	"github.com/machinefi/sprout/task"
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

	// repeat 3 and duration 5s
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
		map[vm.Type]string{
			vm.Risc0:  conf.Risc0ServerEndpoint,
			vm.Halo2:  conf.Halo2ServerEndpoint,
			vm.ZKwasm: conf.ZKWasmServerEndpoint,
		},
	)

	projectManager, err := project.NewManager(conf.ChainEndpoint, conf.ProjectContractAddress, conf.ProjectFileDirectory, conf.ProjectCacheDirectory, conf.IPFSEndpoint)
	if err != nil {
		log.Fatal(err)
	}

	taskProcessor, err := task.NewProcessor(vmHandler, projectManager)
	if err != nil {
		log.Fatal(err)
	}

	_, err = p2p.NewPubSubs(projectManager, taskProcessor, conf.BootNodeMultiAddr, conf.IoTeXChainID)
	if err != nil {
		log.Fatal(err)
	}

	slog.Info("znode started")
}

func runEnode(conf *enodeconfig.Config) {
	pg, err := persistence.NewPostgres(conf.DatabaseDSN)
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

	nextTaskID, err := pg.FetchNextTaskID()
	if err != nil {
		log.Fatal(err)
	}

	dispatcher, err := task.NewDispatcher(pg, projectManager, conf.OperatorPrivateKey, conf.OperatorPrivateKeyED25519)
	if err != nil {
		log.Fatal(err)
	}

	networking, err := p2p.NewPubSubs(projectManager, dispatcher, conf.BootNodeMultiAddr, conf.IoTeXChainID)
	if err != nil {
		log.Fatal(err)
	}

	go task.Dispatching(nextTaskID, datasource, networking)

	go func() {
		if err := api.NewHttpServer(pg, conf).Run(conf.ServiceEndpoint); err != nil {
			log.Fatal(err)
		}
	}()

	slog.Info("znode started")
}
