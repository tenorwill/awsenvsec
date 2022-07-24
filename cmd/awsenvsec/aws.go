package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

// AWS SDK V2: Decrypt secret value - interface and functions
type GetSecretValueAPI interface {
	GetSecretValue(ctx context.Context,
		params *secretsmanager.GetSecretValueInput,
		optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
}

func GetSecret(c context.Context, api GetSecretValueAPI,
	input *secretsmanager.GetSecretValueInput) (*secretsmanager.GetSecretValueOutput, error) {
	return api.GetSecretValue(c, input)
}

func decryptSecret(decryptInput string) (decryptedSecret string) {

	var client *secretsmanager.Client

	// If no profile is specified, assume env variables are set or instance role,
	// otherwise use specified profile:
	if len(*profileFlag) == 0 {
		cfg, err := config.LoadDefaultConfig(context.TODO(),
			config.WithRegion(*regionFlag))
		if err != nil {
			panic("configuration error, " + err.Error())
		}
		client = secretsmanager.NewFromConfig(cfg)
	} else {
		cfg, err := config.LoadDefaultConfig(context.TODO(),
			config.WithRegion(*regionFlag),
			config.WithSharedConfigProfile(*profileFlag))
		if err != nil {
			panic("configuration error, " + err.Error())
		}
		client = secretsmanager.NewFromConfig(cfg)
	}

	input := &secretsmanager.GetSecretValueInput{
		SecretId: &decryptInput,
	}

	result, err := GetSecret(context.TODO(), client, input)
	if err != nil {
		fmt.Println("There was a problem decrypting the secret from AWS Secrets Manager:")
		fmt.Println(err)
		return
	}
	return string(*result.SecretString)
}

// AWS SDK V2: List secrets - interface and functions
type ListSecretsAPIClient interface {
	ListSecrets(ctx context.Context,
		params *secretsmanager.ListSecretsInput,
		optFns ...func(*secretsmanager.Options)) (*secretsmanager.ListSecretsOutput, error)
}

func ListSecrets(c context.Context, api ListSecretsAPIClient,
	input *secretsmanager.ListSecretsInput) (*secretsmanager.ListSecretsOutput, error) {
	return api.ListSecrets(c, input)
}

type SMSecrets struct {
	Name string
}

func getAllSecrets() (outputSecrets []SMSecrets) {

	var client *secretsmanager.Client

	// If no profile is specified, assume env variables are set or instance role,
	// otherwise use specified profile:
	if len(*profileFlag) == 0 {
		cfg, err := config.LoadDefaultConfig(context.TODO(),
			config.WithRegion(*regionFlag))
		if err != nil {
			panic("configuration error, " + err.Error())
		}
		client = secretsmanager.NewFromConfig(cfg)
	} else {
		cfg, err := config.LoadDefaultConfig(context.TODO(),
			config.WithRegion(*regionFlag),
			config.WithSharedConfigProfile(*profileFlag))
		if err != nil {
			panic("configuration error, " + err.Error())
		}
		client = secretsmanager.NewFromConfig(cfg)
	}

	// Filter based on specified AWS Secrets Manager Path
	input := &secretsmanager.ListSecretsInput{
		Filters: []types.Filter{
			{
				Key: types.FilterNameStringTypeName,
				Values: []string{
					*smPathFlag,
				},
			},
		},
	}

	paginator := secretsmanager.NewListSecretsPaginator(client, input, func(o *secretsmanager.ListSecretsPaginatorOptions) {
		o.Limit = 10
		o.StopOnDuplicateToken = false
	})
	pageNum := 0
	var secretResults []SMSecrets
	for paginator.HasMorePages() && pageNum < 50 {

		result, err := paginator.NextPage(context.TODO())
		if err != nil {
			fmt.Println("There was a problem decrypting the secret from Secrets Manager:")
			fmt.Println(err)
			return
		}

		for _, r := range result.SecretList {
			secretResultsSlice := SMSecrets{
				Name: *r.Name,
			}
			secretResults = append(secretResults, secretResultsSlice)
		}
		pageNum++
	}
	return secretResults
}

// Get Parameters By Path
type GetParametersByPathAPI interface {
	GetParametersByPath(ctx context.Context,
		params *ssm.GetParametersByPathInput,
		optFns ...func(*ssm.Options)) (*ssm.GetParametersByPathOutput, error)
}

func GetParametersByPath(c context.Context, api GetParametersByPathAPI,
	input *ssm.GetParametersByPathInput) (*ssm.GetParametersByPathOutput, error) {
	return api.GetParametersByPath(c, input)
}

type PSSecrets struct {
	Name  string
	Value string
}

func getAllParameters() (outputParams []PSSecrets) {

	var client *ssm.Client

	// If no profile is specified, assume env variables are set or instance role,
	// otherwise use specified profile:
	if len(*profileFlag) == 0 {
		cfg, err := config.LoadDefaultConfig(context.TODO(),
			config.WithRegion(*regionFlag))
		if err != nil {
			panic("configuration error, " + err.Error())
		}
		client = ssm.NewFromConfig(cfg)
	} else {
		cfg, err := config.LoadDefaultConfig(context.TODO(),
			config.WithRegion(*regionFlag),
			config.WithSharedConfigProfile(*profileFlag))
		if err != nil {
			panic("configuration error, " + err.Error())
		}
		client = ssm.NewFromConfig(cfg)
	}

	input := &ssm.GetParametersByPathInput{
		Path:           psPathFlag,
		WithDecryption: *aws.Bool(true),
		Recursive:      *recursiveFlag,
	}

	paginator := ssm.NewGetParametersByPathPaginator(client, input, func(o *ssm.GetParametersByPathPaginatorOptions) {
		o.Limit = 10
		o.StopOnDuplicateToken = false
	})

	var parameterResults []PSSecrets
	pageNum := 0
	for paginator.HasMorePages() && pageNum < 50 {

		result, err := paginator.NextPage(context.TODO())
		if err != nil {
			fmt.Println("There was a problem retrieving decrypted secrets from SSM Paramater Store:")
			fmt.Println(err)
			return
		}

		for _, r := range result.Parameters {
			parameterResultsSlice := PSSecrets{
				Name:  *r.Name,
				Value: *r.Value,
			}
			parameterResults = append(parameterResults, parameterResultsSlice)
		}
		pageNum++
	}

	return parameterResults
}
