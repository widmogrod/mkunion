package localstackutil

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"os"
)

func GetEnvOr(key string, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func LoadLocalStackAwsConfig(ctx context.Context) (aws.Config, error) {
	return config.LoadDefaultConfig(
		ctx,
		config.WithRegion(GetEnvOr("AWS_DEFAULT_REGION", "eu-east-1")),
		config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{
				URL:           GetEnvOr("AWS_ENDPOINT_URL", "http://localhost:4566"),
				SigningRegion: region,
			}, nil
		})),
	)
}
