package credentials

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadGCP(t *testing.T) {
	log := logger.Nop()
	loader := NewLoader(log)
	ctx := context.Background()

	// Create temporary GCP credentials file
	tmpFile, err := os.CreateTemp("", "gcp-sa-*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	// Use placeholder private key to avoid security scanner warnings
	gcpJSON := `{
		"type": "service_account",
		"project_id": "test-project",
		"private_key_id": "test-key-id",
		"private_key": "-----BEGIN RSA PRIVATE KEY-----\nTEST-PLACEHOLDER\n-----END RSA PRIVATE KEY-----",
		"client_email": "test@test-project.iam.gserviceaccount.com",
		"client_id": "test-client-id",
		"auth_uri": "https://accounts.google.com/o/oauth2/auth",
		"token_uri": "https://oauth2.googleapis.com/token",
		"auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
		"client_x509_cert_url": "https://www.googleapis.com/robot/v1/metadata/x509/test"
	}`
	_, err = tmpFile.WriteString(gcpJSON)
	require.NoError(t, err)
	tmpFile.Close()

	// Test loading
	creds, err := loader.LoadGCP(ctx, tmpFile.Name())
	require.NoError(t, err)
	assert.Equal(t, "test-project", creds.ProjectID)
	assert.Equal(t, "test@test-project.iam.gserviceaccount.com", creds.ClientEmail)
}

func TestLoadGCP_InvalidJSON(t *testing.T) {
	log := logger.Nop()
	loader := NewLoader(log)
	ctx := context.Background()

	// Create temporary invalid JSON file
	tmpFile, err := os.CreateTemp("", "gcp-sa-*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString("{invalid json")
	require.NoError(t, err)
	tmpFile.Close()

	// Test loading should fail
	_, err = loader.LoadGCP(ctx, tmpFile.Name())
	assert.Error(t, err)
}

func TestLoadAWS_FromFile(t *testing.T) {
	log := logger.Nop()
	loader := NewLoader(log)
	ctx := context.Background()

	// Generate dynamic credentials to avoid security scanner warnings
	defaultAccessKey := fmt.Sprintf("AKIA%s", uuid.New().String()[:16])
	defaultSecretKey := uuid.New().String() + uuid.New().String()[:8]
	testAccessKey := fmt.Sprintf("AKIA%s", uuid.New().String()[:16])
	testSecretKey := uuid.New().String() + uuid.New().String()[:8]
	testSessionToken := uuid.New().String()

	// Create temporary AWS credentials file
	tmpFile, err := os.CreateTemp("", "aws-creds-*")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	awsINI := fmt.Sprintf(`[default]
aws_access_key_id = %s
aws_secret_access_key = %s
region = us-east-1

[test-profile]
aws_access_key_id = %s
aws_secret_access_key = %s
aws_session_token = %s
region = us-west-2
`, defaultAccessKey, defaultSecretKey, testAccessKey, testSecretKey, testSessionToken)

	_, err = tmpFile.WriteString(awsINI)
	require.NoError(t, err)
	tmpFile.Close()

	t.Run("default profile", func(t *testing.T) {
		creds, err := loader.LoadAWS(ctx, AWSCredentialOptions{
			CredentialsFile: tmpFile.Name(),
			Profile:         "default",
		})
		require.NoError(t, err)
		assert.Equal(t, defaultAccessKey, creds.AccessKeyID)
		assert.Equal(t, defaultSecretKey, creds.SecretAccessKey)
		assert.Equal(t, "us-east-1", creds.Region)
		assert.Empty(t, creds.SessionToken)
	})

	t.Run("test-profile with session token", func(t *testing.T) {
		creds, err := loader.LoadAWS(ctx, AWSCredentialOptions{
			CredentialsFile: tmpFile.Name(),
			Profile:         "test-profile",
		})
		require.NoError(t, err)
		assert.Equal(t, testAccessKey, creds.AccessKeyID)
		assert.Equal(t, testSecretKey, creds.SecretAccessKey)
		assert.Equal(t, "us-west-2", creds.Region)
		assert.Equal(t, testSessionToken, creds.SessionToken)
	})

	t.Run("empty profile defaults to default", func(t *testing.T) {
		creds, err := loader.LoadAWS(ctx, AWSCredentialOptions{
			CredentialsFile: tmpFile.Name(),
		})
		require.NoError(t, err)
		assert.Equal(t, defaultAccessKey, creds.AccessKeyID)
	})
}

func TestLoadAWS_FromEnvironment(t *testing.T) {
	log := logger.Nop()
	loader := NewLoader(log)
	ctx := context.Background()

	// Generate dynamic credentials to avoid security scanner warnings
	envAccessKey := fmt.Sprintf("AKIA%s", uuid.New().String()[:16])
	envSecretKey := uuid.New().String() + uuid.New().String()[:8]

	// Set environment variables
	os.Setenv("AWS_ACCESS_KEY_ID", envAccessKey)
	os.Setenv("AWS_SECRET_ACCESS_KEY", envSecretKey)
	os.Setenv("AWS_REGION", "eu-west-1")
	defer func() {
		os.Unsetenv("AWS_ACCESS_KEY_ID")
		os.Unsetenv("AWS_SECRET_ACCESS_KEY")
		os.Unsetenv("AWS_REGION")
	}()

	creds, err := loader.LoadAWS(ctx, AWSCredentialOptions{
		UseEnvironment: true,
	})
	require.NoError(t, err)
	assert.Equal(t, envAccessKey, creds.AccessKeyID)
	assert.Equal(t, envSecretKey, creds.SecretAccessKey)
	assert.Equal(t, "eu-west-1", creds.Region)
}

func TestLoadAWS_FileOverridesEnvironment(t *testing.T) {
	log := logger.Nop()
	loader := NewLoader(log)
	ctx := context.Background()

	// Generate dynamic credentials to avoid security scanner warnings
	fileAccessKey := fmt.Sprintf("AKIA%s", uuid.New().String()[:16])
	fileSecretKey := uuid.New().String() + uuid.New().String()[:8]
	envAccessKey := fmt.Sprintf("AKIA%s", uuid.New().String()[:16])
	envSecretKey := uuid.New().String() + uuid.New().String()[:8]

	// Create temporary AWS credentials file
	tmpFile, err := os.CreateTemp("", "aws-creds-*")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	awsINI := fmt.Sprintf(`[default]
aws_access_key_id = %s
aws_secret_access_key = %s
region = ap-southeast-1
`, fileAccessKey, fileSecretKey)
	_, err = tmpFile.WriteString(awsINI)
	require.NoError(t, err)
	tmpFile.Close()

	// Set environment variables
	os.Setenv("AWS_ACCESS_KEY_ID", envAccessKey)
	os.Setenv("AWS_SECRET_ACCESS_KEY", envSecretKey)
	defer func() {
		os.Unsetenv("AWS_ACCESS_KEY_ID")
		os.Unsetenv("AWS_SECRET_ACCESS_KEY")
	}()

	// File should take precedence
	creds, err := loader.LoadAWS(ctx, AWSCredentialOptions{
		CredentialsFile: tmpFile.Name(),
		UseEnvironment:  true,
	})
	require.NoError(t, err)
	assert.Equal(t, fileAccessKey, creds.AccessKeyID)
	assert.Equal(t, fileSecretKey, creds.SecretAccessKey)
	assert.Equal(t, "ap-southeast-1", creds.Region)
}

func TestLoadAWS_NonExistentProfile(t *testing.T) {
	log := logger.Nop()
	loader := NewLoader(log)
	ctx := context.Background()

	// Generate dynamic credentials to avoid security scanner warnings
	accessKey := fmt.Sprintf("AKIA%s", uuid.New().String()[:16])
	secretKey := uuid.New().String() + uuid.New().String()[:8]

	// Create temporary AWS credentials file
	tmpFile, err := os.CreateTemp("", "aws-creds-*")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	awsINI := fmt.Sprintf(`[default]
aws_access_key_id = %s
aws_secret_access_key = %s
`, accessKey, secretKey)
	_, err = tmpFile.WriteString(awsINI)
	require.NoError(t, err)
	tmpFile.Close()

	// Test loading non-existent profile
	_, err = loader.LoadAWS(ctx, AWSCredentialOptions{
		CredentialsFile: tmpFile.Name(),
		Profile:         "non-existent",
	})
	assert.Error(t, err)
}

func TestLoadAzure_FromFile(t *testing.T) {
	log := logger.Nop()
	loader := NewLoader(log)
	ctx := context.Background()

	// Generate dynamic client secret to avoid security scanner warnings
	clientSecret := uuid.New().String()

	// Create temporary Azure credentials file
	tmpFile, err := os.CreateTemp("", "azure-creds-*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	azureJSON := fmt.Sprintf(`{
		"client_id": "test-client-id",
		"client_secret": "%s",
		"tenant_id": "test-tenant-id"
	}`, clientSecret)
	_, err = tmpFile.WriteString(azureJSON)
	require.NoError(t, err)
	tmpFile.Close()

	creds, err := loader.LoadAzure(ctx, AzureCredentialOptions{
		CredentialsFile: tmpFile.Name(),
	})
	require.NoError(t, err)
	assert.Equal(t, "test-client-id", creds.ClientID)
	assert.Equal(t, clientSecret, creds.ClientSecret)
	assert.Equal(t, "test-tenant-id", creds.TenantID)
}

func TestLoadAzure_FromEnvironment(t *testing.T) {
	log := logger.Nop()
	loader := NewLoader(log)
	ctx := context.Background()

	// Generate dynamic client secret to avoid security scanner warnings
	clientSecret := uuid.New().String()

	// Set environment variables
	os.Setenv("AZURE_CLIENT_ID", "test-client-id")
	os.Setenv("AZURE_CLIENT_SECRET", clientSecret)
	os.Setenv("AZURE_TENANT_ID", "test-tenant-id")
	defer func() {
		os.Unsetenv("AZURE_CLIENT_ID")
		os.Unsetenv("AZURE_CLIENT_SECRET")
		os.Unsetenv("AZURE_TENANT_ID")
	}()

	creds, err := loader.LoadAzure(ctx, AzureCredentialOptions{
		UseEnvironment: true,
	})
	require.NoError(t, err)
	assert.Equal(t, "test-client-id", creds.ClientID)
	assert.Equal(t, clientSecret, creds.ClientSecret)
	assert.Equal(t, "test-tenant-id", creds.TenantID)
}

func TestLoadAzure_FileOverridesEnvironment(t *testing.T) {
	log := logger.Nop()
	loader := NewLoader(log)
	ctx := context.Background()

	// Generate dynamic client secrets to avoid security scanner warnings
	fileClientSecret := uuid.New().String()
	envClientSecret := uuid.New().String()

	// Create temporary Azure credentials file
	tmpFile, err := os.CreateTemp("", "azure-creds-*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	azureJSON := fmt.Sprintf(`{
		"client_id": "file-client-id",
		"client_secret": "%s",
		"tenant_id": "file-tenant-id"
	}`, fileClientSecret)
	_, err = tmpFile.WriteString(azureJSON)
	require.NoError(t, err)
	tmpFile.Close()

	// Set environment variables
	os.Setenv("AZURE_CLIENT_ID", "env-client-id")
	os.Setenv("AZURE_CLIENT_SECRET", envClientSecret)
	os.Setenv("AZURE_TENANT_ID", "env-tenant-id")
	defer func() {
		os.Unsetenv("AZURE_CLIENT_ID")
		os.Unsetenv("AZURE_CLIENT_SECRET")
		os.Unsetenv("AZURE_TENANT_ID")
	}()

	// File should take precedence
	creds, err := loader.LoadAzure(ctx, AzureCredentialOptions{
		CredentialsFile: tmpFile.Name(),
		UseEnvironment:  true,
	})
	require.NoError(t, err)
	assert.Equal(t, "file-client-id", creds.ClientID)
	assert.Equal(t, fileClientSecret, creds.ClientSecret)
	assert.Equal(t, "file-tenant-id", creds.TenantID)
}

func TestLoadAzure_InvalidJSON(t *testing.T) {
	log := logger.Nop()
	loader := NewLoader(log)
	ctx := context.Background()

	// Create temporary invalid JSON file
	tmpFile, err := os.CreateTemp("", "azure-creds-*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString("{invalid json")
	require.NoError(t, err)
	tmpFile.Close()

	_, err = loader.LoadAzure(ctx, AzureCredentialOptions{
		CredentialsFile: tmpFile.Name(),
	})
	assert.Error(t, err)
}

func TestParseAWSCredentialsINI(t *testing.T) {
	// Generate dynamic credentials to avoid security scanner warnings
	defaultAccessKey := fmt.Sprintf("AKIA%s", uuid.New().String()[:16])
	defaultSecretKey := uuid.New().String() + uuid.New().String()[:8]
	defaultSessionToken := uuid.New().String()
	prodAccessKey := fmt.Sprintf("AKIA%s", uuid.New().String()[:16])
	prodSecretKey := uuid.New().String() + uuid.New().String()[:8]

	tests := []struct {
		name        string
		content     string
		profile     string
		expected    *AWSCredentials
		expectError bool
	}{
		{
			name: "basic default profile",
			content: fmt.Sprintf(`[default]
aws_access_key_id = %s
aws_secret_access_key = %s
`, defaultAccessKey, defaultSecretKey),
			profile: "default",
			expected: &AWSCredentials{
				AccessKeyID:     defaultAccessKey,
				SecretAccessKey: defaultSecretKey,
			},
		},
		{
			name: "profile with session token",
			content: fmt.Sprintf(`[default]
aws_access_key_id = %s
aws_secret_access_key = %s
aws_session_token = %s
region = us-west-2
`, defaultAccessKey, defaultSecretKey, defaultSessionToken),
			profile: "default",
			expected: &AWSCredentials{
				AccessKeyID:     defaultAccessKey,
				SecretAccessKey: defaultSecretKey,
				SessionToken:    defaultSessionToken,
				Region:          "us-west-2",
			},
		},
		{
			name: "multiple profiles",
			content: fmt.Sprintf(`[default]
aws_access_key_id = %s
aws_secret_access_key = %s

[prod]
aws_access_key_id = %s
aws_secret_access_key = %s
`, defaultAccessKey, defaultSecretKey, prodAccessKey, prodSecretKey),
			profile: "prod",
			expected: &AWSCredentials{
				AccessKeyID:     prodAccessKey,
				SecretAccessKey: prodSecretKey,
			},
		},
		{
			name: "with comments",
			content: fmt.Sprintf(`# This is a comment
[default]
; Another comment
aws_access_key_id = %s
aws_secret_access_key = %s
`, defaultAccessKey, defaultSecretKey),
			profile: "default",
			expected: &AWSCredentials{
				AccessKeyID:     defaultAccessKey,
				SecretAccessKey: defaultSecretKey,
			},
		},
		{
			name: "empty profile defaults to default",
			content: fmt.Sprintf(`[default]
aws_access_key_id = %s
aws_secret_access_key = %s
`, defaultAccessKey, defaultSecretKey),
			profile: "",
			expected: &AWSCredentials{
				AccessKeyID:     defaultAccessKey,
				SecretAccessKey: defaultSecretKey,
			},
		},
		{
			name: "non-existent profile",
			content: fmt.Sprintf(`[default]
aws_access_key_id = %s
aws_secret_access_key = %s
`, defaultAccessKey, defaultSecretKey),
			profile:     "non-existent",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			creds, err := parseAWSCredentialsINI(tt.content, tt.profile)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected.AccessKeyID, creds.AccessKeyID)
				assert.Equal(t, tt.expected.SecretAccessKey, creds.SecretAccessKey)
				assert.Equal(t, tt.expected.SessionToken, creds.SessionToken)
				assert.Equal(t, tt.expected.Region, creds.Region)
			}
		})
	}
}

func TestRedactPath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "short path",
			input:    "/tmp/file.json",
			expected: "/tmp/file.json",
		},
		{
			name:     "long path",
			input:    "/vault/secrets/very/long/path/to/credentials.json",
			expected: ".../credentials.json",
		},
		{
			name:     "empty path",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := redactPath(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLoadAWS_FromCredentialsFileEnv(t *testing.T) {
	log := logger.Nop()
	loader := NewLoader(log)
	ctx := context.Background()

	// Generate dynamic credentials to avoid security scanner warnings
	accessKey := fmt.Sprintf("AKIA%s", uuid.New().String()[:16])
	secretKey := uuid.New().String() + uuid.New().String()[:8]

	// Create temporary AWS credentials file
	tmpDir := t.TempDir()
	credFile := filepath.Join(tmpDir, "aws-credentials")
	awsINI := fmt.Sprintf(`[default]
aws_access_key_id = %s
aws_secret_access_key = %s
region = ap-northeast-1
`, accessKey, secretKey)
	err := os.WriteFile(credFile, []byte(awsINI), 0600)
	require.NoError(t, err)

	// Set environment variable
	os.Setenv("AWS_CREDENTIALS_FILE", credFile)
	defer os.Unsetenv("AWS_CREDENTIALS_FILE")

	// Load from environment variable
	creds, err := loader.LoadAWS(ctx, AWSCredentialOptions{})
	require.NoError(t, err)
	assert.Equal(t, accessKey, creds.AccessKeyID)
	assert.Equal(t, secretKey, creds.SecretAccessKey)
	assert.Equal(t, "ap-northeast-1", creds.Region)
}

func TestLoadAzure_FromCredentialsFileEnv(t *testing.T) {
	log := logger.Nop()
	loader := NewLoader(log)
	ctx := context.Background()

	// Generate dynamic client secret to avoid security scanner warnings
	clientSecret := uuid.New().String()

	// Create temporary Azure credentials file
	tmpDir := t.TempDir()
	credFile := filepath.Join(tmpDir, "azure-credentials.json")
	azureJSON := fmt.Sprintf(`{
		"client_id": "test-client-id",
		"client_secret": "%s",
		"tenant_id": "test-tenant-id"
	}`, clientSecret)
	err := os.WriteFile(credFile, []byte(azureJSON), 0600)
	require.NoError(t, err)

	// Set environment variable
	os.Setenv("AZURE_CREDENTIALS_FILE", credFile)
	defer os.Unsetenv("AZURE_CREDENTIALS_FILE")

	// Load from environment variable
	creds, err := loader.LoadAzure(ctx, AzureCredentialOptions{})
	require.NoError(t, err)
	assert.Equal(t, "test-client-id", creds.ClientID)
	assert.Equal(t, clientSecret, creds.ClientSecret)
	assert.Equal(t, "test-tenant-id", creds.TenantID)
}
