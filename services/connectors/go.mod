module github.com/quantumlayerhq/ql-rf/services/connectors

go 1.23

require (
	github.com/aws/aws-sdk-go-v2 v1.32.6
	github.com/aws/aws-sdk-go-v2/config v1.28.6
	github.com/aws/aws-sdk-go-v2/credentials v1.17.47
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.194.0
	github.com/aws/aws-sdk-go-v2/service/sts v1.33.2
	github.com/google/uuid v1.6.0
	github.com/quantumlayerhq/ql-rf v0.0.0
	github.com/robfig/cron/v3 v3.0.1
)

replace github.com/quantumlayerhq/ql-rf => ../..
