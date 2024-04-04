package clients

import (
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

var (
	DB    *sqlx.DB
	Redis *redis.Client
)
