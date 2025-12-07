package config

type Config struct {
	Port int `config:"RPC_PORT" default:"6042"`
}
