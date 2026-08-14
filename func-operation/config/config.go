package config

import (
	"github.com/gly-hub/quickgo"
	"github.com/gly-hub/quickgo/db/gorm"
	"github.com/gly-hub/quickgo/db/redis"
	"github.com/gly-hub/quickgo/tracing"
)

type Config struct {
	AppConfig        quickgo.AppConfig        `json:"app" yaml:"app"`
	LoggerConfig     quickgo.LoggerConfig     `json:"logger" yaml:"logger"`
	GrpcServerConfig quickgo.GrpcServerConfig `json:"grpcServer" yaml:"grpcServer"`
	GrpcClientConfig quickgo.GrpcClientConfig `json:"grpcClient" yaml:"grpcClient"`
	GormConfig       gorm.GormManagerConfig   `json:"gorm" yaml:"gorm"`
	RedisConfig      redis.RedisManagerConfig `json:"redis" yaml:"redis"`
	TracingConfig    tracing.Config           `json:"tracing" yaml:"tracing"`
	GeneratedApp     GeneratedAppConfig       `json:"generated_app" yaml:"generated_app"`
}

type GeneratedAppConfig struct {
	RootDir             string                    `json:"root_dir" yaml:"root_dir"`
	ArtifactStore       ArtifactStoreConfig       `json:"artifact_store" yaml:"artifact_store"`
	DataCapability      DataCapabilityConfig      `json:"data_capability" yaml:"data_capability"`
	ArtifactMaintenance ArtifactMaintenanceConfig `json:"artifact_maintenance" yaml:"artifact_maintenance"`
}

type DataCapabilityConfig struct {
	DefaultLimit     int `json:"default_limit" yaml:"default_limit"`
	MaxLimit         int `json:"max_limit" yaml:"max_limit"`
	MaxJoinDepth     int `json:"max_join_depth" yaml:"max_join_depth"`
	MaxResponseBytes int `json:"max_response_bytes" yaml:"max_response_bytes"`
	InvokeTimeoutMs  int `json:"invoke_timeout_ms" yaml:"invoke_timeout_ms"`
}

type ArtifactMaintenanceConfig struct {
	ReconcileIntervalSeconds int `json:"reconcile_interval_seconds" yaml:"reconcile_interval_seconds"`
	StaleStagingSeconds      int `json:"stale_staging_seconds" yaml:"stale_staging_seconds"`
}

type ArtifactStoreConfig struct {
	Driver string                `json:"driver" yaml:"driver"`
	S3     S3ArtifactStoreConfig `json:"s3" yaml:"s3"`
}

type S3ArtifactStoreConfig struct {
	Endpoint  string `json:"endpoint" yaml:"endpoint"`
	AccessKey string `json:"access_key" yaml:"access_key"`
	SecretKey string `json:"secret_key" yaml:"secret_key"`
	Bucket    string `json:"bucket" yaml:"bucket"`
	Prefix    string `json:"prefix" yaml:"prefix"`
	Region    string `json:"region" yaml:"region"`
	UseSSL    bool   `json:"use_ssl" yaml:"use_ssl"`
}
