package mariadb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

const (
	dockerTimeout  = 15 * time.Second
	startupTimeout = 3 * time.Minute
	readyTimeout   = 90 * time.Second
	pingInterval   = 500 * time.Millisecond
)

var errDockerUnavailable = errors.New("Docker CLI or daemon is unavailable")

// Server connects only to a loopback port published by its disposable container.
// There is deliberately no external DSN or image override.
type Server struct {
	container string
	admin     *sqlx.DB
	config    *mysql.Config
}

func startServer() (_ *Server, err error) {
	if _, err := docker(dockerTimeout, "info", "--format", "{{.ServerVersion}}"); err != nil {
		return nil, fmt.Errorf("%w: %v", errDockerUnavailable, err)
	}
	image, err := pinnedImage()
	if err != nil {
		return nil, err
	}
	name, err := randomName("dflh-test-")
	if err != nil {
		return nil, err
	}
	password, err := randomName("")
	if err != nil {
		return nil, err
	}
	s := &Server{container: name}
	defer func() {
		if err != nil {
			err = errors.Join(err, s.close())
		}
	}()
	if _, err := docker(startupTimeout, "run", "-d", "--name", name,
		"--platform", "linux/amd64", "-e", "MYSQL_ROOT_PASSWORD="+password,
		"-p", "127.0.0.1::3306", image); err != nil {
		return nil, err
	}
	address, err := docker(dockerTimeout, "port", name, "3306/tcp")
	if err != nil {
		return nil, err
	}
	host, _, err := net.SplitHostPort(address)
	if err != nil || host != "127.0.0.1" {
		return nil, fmt.Errorf("unexpected Docker test address %q", address)
	}
	config := mysql.NewConfig()
	config.User, config.Passwd = "root", password
	config.Net, config.Addr = "tcp", address
	config.ParseTime, config.MultiStatements = true, true
	config.Timeout = time.Second
	config.ReadTimeout, config.WriteTimeout = queryTimeout, queryTimeout
	config.Params = map[string]string{"charset": "utf8mb4"}
	s.config = config
	// Readiness probes can see EOF while the image initializes its data directory.
	adminConfig := config.Clone()
	adminConfig.Logger = log.New(io.Discard, "", 0)
	connector, err := mysql.NewConnector(adminConfig)
	if err != nil {
		return nil, err
	}
	s.admin = sqlx.NewDb(sql.OpenDB(connector), "mysql")
	deadline := time.Now().Add(readyTimeout)
	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		err = s.admin.PingContext(ctx)
		cancel()
		if err == nil {
			var version string
			ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
			err = s.admin.GetContext(ctx, &version, "SELECT VERSION()")
			cancel()
			if err != nil {
				return nil, err
			}
			if version != "10.1.38" && !strings.HasPrefix(version, "10.1.38-") {
				return nil, fmt.Errorf("unexpected MariaDB version %q", version)
			}
			return s, nil
		}
		time.Sleep(pingInterval)
	}
	return nil, fmt.Errorf("MariaDB did not become ready within %s: %w", readyTimeout, err)
}

func (s *Server) close() error {
	var closeErr error
	if s.admin != nil {
		closeErr = s.admin.Close()
	}
	_, removeErr := docker(dockerTimeout, "rm", "-fv", s.container)
	return errors.Join(closeErr, removeErr)
}

func docker(timeout time.Duration, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	output, err := exec.CommandContext(ctx, "docker", args...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("docker %s: %w (%s)", args[0], err, strings.TrimSpace(string(output)))
	}
	return strings.TrimSpace(string(output)), nil
}

func pinnedImage() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			data, err := os.ReadFile(filepath.Join(dir, "migrations", "testdata", "mariadb-10.1.38.image"))
			if err != nil {
				return "", err
			}
			image := strings.TrimSpace(string(data))
			if !strings.HasPrefix(image, "mariadb@sha256:") {
				return "", fmt.Errorf("expected pinned MariaDB image digest")
			}
			return image, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("cannot find backend go.mod for pinned MariaDB image")
		}
		dir = parent
	}
}
