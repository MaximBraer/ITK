package postgres

import (
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DBConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	Database        string
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxLifetime int
}

func (c *DBConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		c.User,
		c.Password,
		c.Host,
		c.Port,
		c.Database,
	)
}

func (c *DBConfig) PoolConfig() (*pgxpool.Config, error) {
	config, err := pgxpool.ParseConfig(c.DSN())
	if err != nil {
		return nil, err
	}

	config.MaxConns = int32(c.MaxOpenConns)
	config.MinConns = int32(c.MaxIdleConns)
	config.MaxConnLifetime = time.Duration(c.ConnMaxLifetime) * time.Second

	return config, nil
}
