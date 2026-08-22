package config

import (
	uploader_minio "github.com/gly-hub/ai-dandelion/toolbox/uploader-minio"
	"github.com/gly-hub/quickgo"
	"github.com/gly-hub/quickgo/db/gorm"
	"github.com/gly-hub/quickgo/db/redis"
	"github.com/gly-hub/quickgo/tracing"
)

type Config struct {
	AppConfig        quickgo.AppConfig          `json:"app" yaml:"app"`
	LoggerConfig     quickgo.LoggerConfig       `json:"logger" yaml:"logger"`
	GrpcServerConfig quickgo.GrpcServerConfig   `json:"grpcServer" yaml:"grpcServer"`
	GormConfig       gorm.GormManagerConfig     `json:"gorm" yaml:"gorm"`
	RedisConfig      redis.RedisManagerConfig   `json:"redis" yaml:"redis"`
	TracingConfig    tracing.Config             `json:"tracing" yaml:"tracing"`
	AuthConfig       AuthConfig                 `json:"auth" yaml:"auth"`
	UploaderConfig   uploader_minio.MinioConfig `json:"uploader" yaml:"uploader"`
}

type AuthConfig struct {
	AccessTokenSecret string `json:"access_token_secret" yaml:"access_token_secret"`
	AccessTokenTTL    string `json:"access_token_ttl" yaml:"access_token_ttl"`
	RefreshTokenTTL   string `json:"refresh_token_ttl" yaml:"refresh_token_ttl"`
}
