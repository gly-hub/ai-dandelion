package config

import (
	"github.com/gly-hub/quickgo"
	"github.com/gly-hub/quickgo/db/mongodb"
	"github.com/gly-hub/quickgo/db/redis"
	"github.com/gly-hub/quickgo/tracing"
)

type Config struct {
	AppConfig        quickgo.AppConfig          `json:"app" yaml:"app"`
	LoggerConfig     quickgo.LoggerConfig       `json:"logger" yaml:"logger"`
	GrpcClientConfig quickgo.GrpcClientConfig   `json:"grpcClient" yaml:"grpcClient"`
	HttpServerConfig quickgo.HTTPServerConfig   `json:"httpServer" yaml:"httpServer"`
	RedisConfig      redis.RedisManagerConfig   `json:"redis" yaml:"redis"`
	MongoDBConfig    mongodb.MongoManagerConfig `json:"mongodb" yaml:"mongodb"`
	TracingConfig    tracing.Config             `json:"tracing" yaml:"tracing"`
	AuthConfig       AuthConfig                 `json:"auth" yaml:"auth"`
	RealtimeConfig   RealtimeConfig             `json:"realtime" yaml:"realtime"`
}

type AuthConfig struct {
	TokenSecret string `json:"token_secret" yaml:"token_secret"`
	TokenTTL    string `json:"token_ttl" yaml:"token_ttl"`
	BridgeToken string `json:"bridge_token" yaml:"bridge_token"`
}

type RealtimeConfig struct {
	AllowedOrigins []string `json:"allowed_origins" yaml:"allowedOrigins"`
}
