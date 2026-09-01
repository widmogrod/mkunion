module github.com/widmogrod/mkunion

go 1.25.1

retract v0.0.0-20220926122055-0884a4bef836 // Contains replace directives, cannot be installed

require (
	github.com/alecthomas/participle/v2 v2.1.4
	github.com/aws/aws-sdk-go-v2 v1.45.1
	github.com/aws/aws-sdk-go-v2/config v1.31.16
	github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue v1.20.13
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.65.1
	github.com/aws/aws-sdk-go-v2/service/kinesis v1.41.1
	github.com/aws/aws-sdk-go-v2/service/sqs v1.42.12
	github.com/confluentinc/confluent-kafka-go/v2 v2.12.0
	github.com/fsnotify/fsnotify v1.9.0
	github.com/google/go-cmp v0.7.0
	github.com/opensearch-project/opensearch-go/v2 v2.3.0
	github.com/robfig/cron/v3 v3.0.1
	github.com/sashabaranov/go-openai v1.41.2
	github.com/sirupsen/logrus v1.9.3
	github.com/stretchr/testify v1.10.0
	github.com/urfave/cli/v2 v2.27.7
	golang.org/x/exp v0.0.0-20250506013437-ce4c2cf36ca6
	golang.org/x/mod v0.24.0
)

require (
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.7.2 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.18.20 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.12 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.5.1 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.8.1 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/dynamodbstreams v1.31.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.19 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/endpoint-discovery v1.13.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.12 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.30.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.35.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.39.0 // indirect
	github.com/aws/smithy-go v1.28.1 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.7 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/matryer/moq v0.6.0 // indirect
	github.com/niemeyer/pretty v0.0.0-20200227124842-a10e7caefd8e // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/xrash/smetrics v0.0.0-20240521201337-686a1a2994c1 // indirect
	golang.org/x/crypto v0.38.0 // indirect
	golang.org/x/sync v0.14.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/tools v0.33.0 // indirect
	gopkg.in/check.v1 v1.0.0-20200227125254-8fa46927fb4f // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

tool (
	github.com/matryer/moq
	github.com/widmogrod/mkunion/cmd/mkfunc
	github.com/widmogrod/mkunion/cmd/mkunion
)
