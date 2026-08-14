package uploader_minio

type MinioConfig struct {
	Address         string `json:"address" yaml:"address"`
	AccessKeyId     string `json:"accessKeyId" yaml:"accessKeyId"`
	SecretAccessKey string `json:"secretAccessKey" yaml:"secretAccessKey"`
	Secure          bool   `json:"secure" yaml:"secure"`
	Bucket          string `json:"bucket" yaml:"bucket"`
	BasePath        string `json:"basePath" yaml:"basePath"`
	Location        string `json:"location" yaml:"location"`
	Domain          string `json:"domain" yaml:"domain"`
}
